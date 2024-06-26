package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

var style = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	//Background(lipgloss.Color("#e785ff")).
	Background(lipgloss.Color("#24001e")).
	PaddingTop(0).
	PaddingBottom(0).
	PaddingLeft(0).
	PaddingRight(0)
	//Width(80)

var field string
var interval int
var ease int
var reps int
var notesId int
var reviewTime int
var reviewDate time.Time

var idx int

var speedScale int64 = 50

type responseMsg struct{}

func getNextReviewTime(key string, m map[string][]int) int {
	var reviewTime int
	if len(m[key]) > 1 {
		reviewTime = m[key][1] - m[key][0]
		m[key] = m[key][1:]
	} else {
		return -10
	}
	return reviewTime
}

func (m model) updateVocab(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		for {

			time.Sleep(time.Duration(1000/speedScale) * time.Millisecond)
			sub <- struct{}{}
		}
	}
}

// A command that waits for the activity on a channel.
func waitForActivity(sub chan struct{}) tea.Cmd {
	return func() tea.Msg {
		return responseMsg(<-sub)
	}
}

type model struct {
	sub                chan struct{} // where we'll receive activity notifications
	triggerActivity    int           // iterate on this value to trigger update through channel
	userInterfaceIdx   int           // current idx of the UI/view
	quitting           bool
	vocabSlice         []string
	vocabIdxMap        map[string][]int // stores the idx of where particular vocabs occurs in the vocab slice
	userInterfaceMap   map[string]int   // store idx of where vocab is on the UI/view
	userInterfaceSlice []vocab          // order of vocab in the UI/view
}

type vocab struct {
	fieldStr       string // unparsed fields for anki card
	nextReviewTime int    // time until next review in ms
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.updateVocab(m.sub),
		waitForActivity(m.sub), // wait for activity
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit
	case responseMsg:
		key := m.vocabSlice[m.triggerActivity]

		_, fnd := m.userInterfaceMap[key]
		if fnd {
			m.userInterfaceSlice[m.userInterfaceMap[key]].nextReviewTime = getNextReviewTime(key, m.vocabIdxMap)
		} else {
			m.userInterfaceSlice = append(m.userInterfaceSlice, vocab{fieldStr: key, nextReviewTime: -1}) // new vocab
			m.userInterfaceMap[key] = m.userInterfaceIdx
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
	s := fmt.Sprintln(style.Render("anki visualizer\n"))

	// sort by latest vocab on the bottom
	for _, v := range m.userInterfaceSlice {
		s += fmt.Sprintln(style.Render(fieldParser(v.fieldStr)[7], fieldParser(v.fieldStr)[4], strconv.Itoa(v.nextReviewTime)))
	}

	// sort by oldest vocab on the bottom
	for i := len(m.userInterfaceSlice) - 1; i >= 0; i-- {
		s += fmt.Sprintln(style.Render(fieldParser(m.userInterfaceSlice[i].fieldStr)[7], fieldParser(m.userInterfaceSlice[i].fieldStr)[4], strconv.Itoa(m.userInterfaceSlice[i].nextReviewTime)))
	}

	if m.quitting {
		s += "\n"
	}
	return s
}

func fieldParser(str string) []string {
	// define delimiter from anki deck (0x1F)
	delimiter := string(rune(0x1F))

	// split the string using the delimiter
	parts := strings.Split(str, delimiter)
	// japanese to english
	// indexes
	// 2 = kanji
	// 3 = hiragana
	// 4 = english translation
	// 7 = kanji + hiragana

	//for i, part := range parts {
	//	fmt.Printf("Part %d: %s\n", i+1, part)
	//}

	return parts
}

func calcDuration(duration int) {
	//	var timeInMs int
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

	m := make(map[string][]int)
	var s []string

	var vocabStr string

	idx = 0

	for rows.Next() {
		err := rows.Scan(&field, &interval, &ease, &reps, &notesId, &reviewTime)
		if err != nil {
			log.Fatal(err)
		}

		fieldSlice := fieldParser(field)
		if len(fieldSlice) > 24 { //filter out reviews not from the core10k deck
			//fmt.Println(style.Render("cards.nid", strconv.Itoa(notesId), fieldSlice[7], fieldSlice[4], "cards.ivl", strconv.Itoa(interval), "revlog.ease", strconv.Itoa(ease), "cards.reps", strconv.Itoa(reps)))
			//reviewDate = time.UnixMilli(int64(reviewTime))
			//fmt.Println(reviewDate, reviewDate.Day(), reviewTime)

			s = append(s, field)

			//vocabStr = fieldSlice[7]
			vocabStr = field

			_, found := m[vocabStr]
			if found {
				m[vocabStr] = append(m[vocabStr], idx)
			} else {
				m[vocabStr] = make([]int, 1)
				m[vocabStr][0] = idx
			}

			idx++
		} else {
			//fmt.Println(style.Render("cards.nid", strconv.Itoa(notesId), fieldSlice[0], fieldSlice[0], "cards.ivl", strconv.Itoa(interval), "revlog.ease", strconv.Itoa(ease), "cards.reps", strconv.Itoa(reps), "cards.type", strconv.Itoa(cardType)))
		}
	}

	//	for _, v := range s {
	//		fmt.Println(fieldParser(v)[7])
	//	}

	//	for i := len(s) - 1; i >= 0; i-- {
	//		//fmt.Println(fieldParser(s[i])[7])
	//		fmt.Println(s[i])
	//	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	//	for k, v := range m {
	//		fmt.Printf(k)
	//		for _, idxs := range v {
	//			fmt.Printf("%s", strconv.Itoa(idxs))
	//			fmt.Printf("\n")
	//		}
	//	}
	//	for k, v := range m {
	//		fmt.Printf(k)
	//		for _, idxs := range v {
	//			fmt.Printf("%s", strconv.Itoa(idxs))
	//			fmt.Printf("\n")
	//			fmt.Println("NEXT DURATION", getNextReviewTime(k, m))
	//		}
	//	}

	p := tea.NewProgram(model{
		sub:                make(chan struct{}),
		vocabSlice:         s,
		vocabIdxMap:        m,
		userInterfaceIdx:   1,
		triggerActivity:    0,
		userInterfaceMap:   map[string]int{s[0]: 0},
		userInterfaceSlice: []vocab{{s[0], 1}},
		//userInterfaceMap:   make(map[string]int),
		//userInterfaceSlice: make([]vocab, 0),
	})

	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
