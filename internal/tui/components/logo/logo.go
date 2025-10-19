// Package logo renders a Tulpa wordmark in a stylized way.
package logo

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/tulpa-code/tulpa/internal/tui/styles"
	"github.com/tulpa-code/tulpa/internal/version"
)

// Opts are the options for rendering the Tulpa title art.
type Opts struct {
	FieldColor    color.Color // diagonal lines
	TitleColorA   color.Color // left gradient ramp point
	TitleColorB   color.Color // right gradient ramp point
	SubtitleColor color.Color // Subtitle text color
	VersionColor  color.Color // Version text color
	Width         int         // width of the rendered logo, used for truncation
}

// Render renders the Tulpa logo. Set the argument to true to render the narrow
// version, intended for use in a sidebar.
func Render(version string, compact bool, o Opts) string {
	const tulpaTag = " Stay Focused"

	fg := func(c color.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Render(s)
	}

	// Simple ASCII art for TULPA
	asciiArt := `
░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓███████▓▒░ ░▒▓██████▓▒░
   ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓███████▓▒░░▒▓████████▓▒░
   ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░   ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░
   ░▒▓█▓▒░    ░▒▓██████▓▒░░▒▓████████▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░
`

	// Remove leading/trailing whitespace and split into lines
	lines := strings.Split(strings.TrimSpace(asciiArt), "\n")

	// Apply gradient to each line
	var gradientLines []string
	for _, line := range lines {
		gradientLines = append(gradientLines, styles.ApplyForegroundGrad(line, o.TitleColorA, o.TitleColorB))
	}

	tulpa := strings.Join(gradientLines, "\n")
	tulpaWidth := lipgloss.Width(lines[0]) // width of first line

	// Tulpa and version.
	metaRowGap := 1
	maxVersionWidth := tulpaWidth - lipgloss.Width(tulpaTag) - metaRowGap
	version = ansi.Truncate(version, maxVersionWidth, "…") // truncate version if too long.
	gap := max(0, tulpaWidth-lipgloss.Width(tulpaTag)-lipgloss.Width(version))
	metaRow := fg(o.SubtitleColor, tulpaTag) + strings.Repeat(" ", gap) + fg(o.VersionColor, version)

	// Join the meta row and big Tulpa title.
	tulpa = strings.TrimSpace(metaRow + "\n" + tulpa)

	// Narrow version.
	if compact {
		field := fg(o.FieldColor, strings.Repeat("#", tulpaWidth))
		return strings.Join([]string{field, field, tulpa, field, ""}, "\n")
	}

	fieldHeight := lipgloss.Height(tulpa)

	// Left field.
	const leftWidth = 6
	leftFieldRow := fg(o.FieldColor, strings.Repeat("#", leftWidth))
	leftField := new(strings.Builder)
	for range fieldHeight {
		fmt.Fprintln(leftField, leftFieldRow)
	}

	// Right field.
	rightWidth := max(15, o.Width-tulpaWidth-leftWidth-2) // 2 for the gap.
	const stepDownAt = 0
	rightField := new(strings.Builder)
	for i := range fieldHeight {
		width := rightWidth
		if i >= stepDownAt {
			width = rightWidth - (i - stepDownAt)
		}
		fmt.Fprint(rightField, fg(o.FieldColor, strings.Repeat("#", width)), "\n")
	}

	// Return the wide version.
	const hGap = " "
	logo := lipgloss.JoinHorizontal(lipgloss.Top, leftField.String(), hGap, tulpa, hGap, rightField.String())
	if o.Width > 0 {
		// Truncate the logo to the specified width.
		lines := strings.Split(logo, "\n")
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, o.Width, "")
		}
		logo = strings.Join(lines, "\n")
	}
	return logo
}

// SmallRender renders a smaller version of the Tulpa logo, suitable for
// smaller windows or sidebar usage.
func SmallRender(width int) string {
	t := styles.CurrentTheme()

	// Compact ASCII art for TULPA
	asciiArt := `
░▀█▀░█░█░█░░░█▀█░█▀█
░░█░░█░█░█░░░█▀▀░█▀█
░░▀░░▀▀▀░▀▀▀░▀░░░▀░▀
`

	// Remove leading/trailing whitespace and split into lines
	lines := strings.Split(strings.TrimSpace(asciiArt), "\n")

	// Apply gradient to each line
	var gradientLines []string
	for _, line := range lines {
		gradientLines = append(gradientLines, styles.ApplyForegroundGrad(line, t.Secondary, t.Primary))
	}

	// Add version on top
	versionText := t.S().Base.Foreground(t.Secondary).Render("v" + version.Version)

	return versionText + "\n" + strings.Join(gradientLines, "\n")
}
