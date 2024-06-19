package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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

	stmt := `select flds, cards.ivl, revlog.ease, cards.reps, cards.nid, revlog.id, cards.type
		from revlog join cards on revlog.cid = cards.id join notes on cards.nid = notes.id
		--where notes.id = 1378555091530
		order by revlog.id;`

	rows, err := db.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var field string
		var interval int
		var ease int
		var reps int
		var notesId int
		var reviewTime int
		var reviewDate time.Time
		var reviewType int
		err := rows.Scan(&field, &interval, &ease, &reps, &notesId, &reviewTime, &reviewType)
		if err != nil {
			log.Fatal(err)
		}

		fieldSlice := fieldParser(field)
		if len(fieldSlice) > 24 {
			//fmt.Println(style.Render(fieldSlice[7], fieldSlice[4], strconv.Itoa(interval)))
			fmt.Println(style.Render("cards.nid", strconv.Itoa(notesId), fieldSlice[7], fieldSlice[4], "cards.ivl", strconv.Itoa(interval), "revlog.ease", strconv.Itoa(ease), "cards.reps", strconv.Itoa(reps), "cards.type", strconv.Itoa(reviewType)))
			reviewDate = time.UnixMilli(int64(reviewTime))
			//fmt.Println(style.Render(fieldSlice[7]))
			fmt.Println(reviewDate, reviewDate.Day())
		} else {
			//fmt.Println(style.Render("cards.nid", strconv.Itoa(notesId), fieldSlice[0], fieldSlice[0], "cards.ivl", strconv.Itoa(interval), "revlog.ease", strconv.Itoa(ease), "cards.reps", strconv.Itoa(reps), "cards.type", strconv.Itoa(cardType)))
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
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
