package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

const ( // anki fields
	Kanji               = 2
	Hiragana            = 3
	KanjiHiragana       = 7
	EngTranslation      = 4
	Core10kCardFieldNum = 24
	AnkiDelimiter       = rune(0x1F)
)

var headerStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	BorderStyle(lipgloss.NormalBorder()).
	PaddingTop(0).
	PaddingBottom(0).
	PaddingLeft(0).
	PaddingRight(0).
	Width(60).
	Align(lipgloss.Center)

var style = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#24001e")).
	PaddingTop(0).
	PaddingBottom(0).
	PaddingLeft(0).
	PaddingRight(0).
	Width(30).
	Align(lipgloss.Center)

var englishStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	PaddingTop(0).
	PaddingBottom(0).
	PaddingLeft(0).
	PaddingRight(0)

type ColourBrightness struct {
	ColourRange []string
	Strength    int
}

func (c *ColourBrightness) IncreaseStrength() string {
	if c.Strength < len(c.ColourRange)-1 {
		c.Strength++
	}
	return c.GetColour()
}

func (c *ColourBrightness) DecreaseStrength() string {
	if c.Strength > 0 {
		c.Strength--
	}
	return c.GetColour()
}

func (c *ColourBrightness) GetColour() string {
	return c.ColourRange[c.Strength]
}

func NewDefaultColour() *ColourBrightness {
	c := new(ColourBrightness)
	//colours := []string{"#24001e",
	//	"#3a0f37",
	//	"#4f1e50",
	//	"#652c69",
	//	"#7b3b82",
	//	"#904a9b",
	//	"#a659b4",
	//	"#bc67cd",
	//	"#d176e6",
	//	"#e785ff"}
	colours := []string{
		"#24001e",
		"#185521",
		"#0caa23",
		"#00ff26"}

	c.ColourRange = colours

	c.Strength = 0
	return c
}

type Card struct {
	Field      string
	Interval   int
	Ease       int
	Reps       int
	NotesId    int
	ReviewTime int
	ReviewDate time.Time
}

var idx int
var speedScale int64 = 50

type responseMsg struct{}

func GetNextReviewTime(key string, m map[string][]int) int {
	var reviewTime int
	if len(m[key]) > 1 {
		reviewTime = m[key][1] - m[key][0]
		m[key] = m[key][1:]
	} else {
		return -10
	}
	return reviewTime
}

func (m model) UpdateVocab(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {
			time.Sleep(time.Duration(1000/speedScale) * time.Millisecond)
			sub <- struct{}{}
		}
	}
}

func waitForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
	}
}

type model struct {
	sub              chan struct{} // where we'll receive activity notifications
	triggerActivity  int           // iterate on this value to trigger update through channel
	userInterfaceIdx int           // current idx of the UI/view
	quitting         bool
	vocab            []string
	vocabIndexes     map[string][]int // stores the idx of where particular vocabs occurs in the vocab slice
	vocabViewIndexes map[string]int   // store idx of where vocab is on the UI/view
	vocabView        []Vocab          // order of vocab in the UI/view
	clrBrightness    *ColourBrightness
}

type Vocab struct {
	FieldStr       string // unparsed fields for anki card
	NextReviewTime int    // time until next review in ms
	Colour         *ColourBrightness
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.UpdateVocab(m.sub),
		waitForActivity(m.sub),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case responseMsg:
		key := m.vocab[m.triggerActivity]

		_, fnd := m.vocabViewIndexes[key]
		if fnd {
			reviewTime := GetNextReviewTime(key, m.vocabIndexes)
			if m.vocabView[m.vocabViewIndexes[key]].NextReviewTime <= reviewTime {
				m.vocabView[m.vocabViewIndexes[key]].Colour.IncreaseStrength()
			} else {
				m.vocabView[m.vocabViewIndexes[key]].Colour.DecreaseStrength()
			}
			m.vocabView[m.vocabViewIndexes[key]].NextReviewTime = reviewTime
		} else {
			m.vocabView = append(m.vocabView, Vocab{FieldStr: key, NextReviewTime: -1, Colour: NewDefaultColour()}) // new vocab
			m.vocabViewIndexes[key] = m.userInterfaceIdx
			//m.vocabView[m.userInterfaceIdx] = Vocab{FieldStr: key, NextReviewTime: -1, Colour: NewDefaultColour()}
			m.userInterfaceIdx++
		}
		m.triggerActivity++
		return m, waitForActivity(m.sub) // wait for next event
	case spinner.TickMsg:
		var cmd tea.Cmd
		return m, cmd
	default:
		return m, nil
	}
}

func (m model) View() string {
	s := fmt.Sprintln(headerStyle.Render("anki visualizer"))
	s += "\n"

	// sort by latest vocab on the bottom
	for _, v := range m.vocabView {
		style = lipgloss.NewStyle().Background(lipgloss.Color(v.Colour.GetColour())).Inherit(style)
		s += fmt.Sprintf(style.Render(fieldParser(v.FieldStr)[KanjiHiragana]))
		s += " " + fmt.Sprintf(englishStyle.Render(fieldParser(v.FieldStr)[EngTranslation])) + "\n"
	}

	//for i := range m.userInterfaceIdx {
	//	style = lipgloss.NewStyle().Background(lipgloss.Color(m.vocabView[i].Colour.GetColour())).Inherit(style)
	//	s += fmt.Sprintf(style.Render(fieldParser(m.vocabView[i].FieldStr)[KanjiHiragana]))
	//	s += " " + fmt.Sprintf(englishStyle.Render(fieldParser(m.vocabView[i].FieldStr)[EngTranslation])) + "\n"
	//}
	// sort by oldest vocab on the bottom
	//for i := len(m.vocabView) - 1; i >= 0; i-- {
	//	style = lipgloss.NewStyle().Background(lipgloss.Color(m.vocabView[i].Colour.GetColour())).Inherit(style)
	//	s += fmt.Sprintf(style.Render(fieldParser(m.vocabView[i].FieldStr)[7], fieldParser(m.vocabView[i].FieldStr)[4], strconv.Itoa(m.vocabView[i].NextReviewTime)))
	//	s += "\n"
	//}

	if m.quitting {
		s += "\n"
	}
	return s
}

func fieldParser(str string) []string {
	// define delimiter from anki deck (0x1F)
	delimiter := string(AnkiDelimiter)

	parts := strings.Split(str, delimiter)

	return parts
}

func IntializeViewSlice(len int, v []string) *[]Vocab {
	vocab := Vocab{FieldStr: v[0], NextReviewTime: 1, Colour: NewDefaultColour()}
	viewVocab := make([]Vocab, len)
	viewVocab[0] = vocab

	return &viewVocab
}

func main() {
	db, err := sql.Open("sqlite3", "./collection.anki2")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Error verifying database connection:", err)
	}
	log.Println("Database connection verified successfully")

	stmt := `select flds, cards.ivl, revlog.ease, cards.reps, cards.nid, revlog.id
		from revlog join cards on revlog.cid = cards.id join notes on cards.nid = notes.id
		--where notes.id = 1378555091530
		order by revlog.id;`

	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	vocabIndexes := make(map[string][]int)
	var vocab []string
	var vocabStr string
	c := &Card{}
	idx = 0

	for rows.Next() {
		err := rows.Scan(&c.Field, &c.Interval, &c.Ease, &c.Reps, &c.NotesId, &c.ReviewTime)
		if err != nil {
			log.Fatal(err)
		}

		fields := fieldParser(c.Field)
		if len(fields) > Core10kCardFieldNum {
			//reviewDate = time.UnixMilli(int64(reviewTime))
			//fmt.Println(reviewDate, reviewDate.Day(), reviewTime)

			vocab = append(vocab, c.Field)

			//vocabStr = fieldSlice[7]
			vocabStr = c.Field

			_, found := vocabIndexes[vocabStr]
			if found {
				vocabIndexes[vocabStr] = append(vocabIndexes[vocabStr], idx)
			} else {
				vocabIndexes[vocabStr] = make([]int, 1)
				vocabIndexes[vocabStr][0] = idx
			}

			idx++
		} else {
			//fmt.Println(style.Render("cards.nid", strconv.Itoa(notesId), fieldSlice[0], fieldSlice[0], "cards.ivl", strconv.Itoa(interval), "revlog.ease", strconv.Itoa(ease), "cards.reps", strconv.Itoa(reps), "cards.type", strconv.Itoa(cardType)))
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(model{
		sub:              make(chan struct{}),
		vocab:            vocab,
		vocabIndexes:     vocabIndexes,
		userInterfaceIdx: 1,
		triggerActivity:  0,
		vocabViewIndexes: map[string]int{vocab[0]: 0},
		vocabView:        []Vocab{{vocab[0], 1, NewDefaultColour()}},
		//vocabView:     *IntializeViewSlice(len(vocabIndexes), vocab),
		clrBrightness: NewDefaultColour(),
	})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
