// Package console provides a set of functions for printing to the console.
package console

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	tabStyle = lipgloss.NewStyle().
			TabWidth(lipgloss.NoTabConversion)

	infoBlock = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true).
			Padding(0, 1).
			Render("INFO")

	warnBlock = lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true).
			Padding(0, 1).
			Render("WARN")

	errorBlock = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true).
			Padding(0, 1).
			Render("ERROR")

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Italic(true)

	questionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Padding(0, 1).
			Bold(true)

	lineStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	cellStyle = lipgloss.NewStyle().
			Padding(0, 1)
)

func Info(msg string, args ...any) {
	fmt.Println(
		tabStyle.Render(
			fmt.Sprintf("%s\t%s", infoBlock, lineStyle.Render(fmt.Sprintf(msg, args...))),
		),
	)
}

func Warn(msg string, args ...any) {
	fmt.Println(
		tabStyle.Render(
			fmt.Sprintf("%s\t%s", warnBlock, lineStyle.Render(fmt.Sprintf(msg, args...))),
		),
	)
}

func Error(msg string, args ...any) {
	fmt.Println(
		tabStyle.Render(
			fmt.Sprintf("%s\t%s", errorBlock, lineStyle.Render(fmt.Sprintf(msg, args...))),
		),
	)
}

func Comment(msg string, args ...any) {
	fmt.Println(commentStyle.Render(fmt.Sprintf(msg, args...)))
}

func Question(msg string, args ...any) {
	fmt.Println(questionStyle.Render(fmt.Sprintf(msg, args...)))
}

func Line(msg string, args ...any) {
	fmt.Println(lineStyle.Render(fmt.Sprintf(msg, args...)))
}

func Table(headers []string, rows [][]string) {
	out := table.New().
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			return cellStyle
		}).
		Headers(headers...).
		Rows(rows...).
		Render()

	fmt.Println(out)
}
