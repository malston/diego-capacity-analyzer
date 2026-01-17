// ABOUTME: Sparkline widget renders mini trend charts using block characters
// ABOUTME: Provides compact visual representation of value history

package widgets

import (
	"github.com/charmbracelet/lipgloss"
)

// SparklineBlocks are the Unicode block characters for different heights
var SparklineBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a compact trend visualization
// values: slice of values to display (most recent last)
// width: number of characters to render (will sample/pad as needed)
// color: optional color for the sparkline
func Sparkline(values []float64, width int, color lipgloss.Color) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	// Sample or pad values to match width
	sampled := sampleValues(values, width)

	// Find min/max for scaling
	min, max := sampled[0], sampled[0]
	for _, v := range sampled {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Build sparkline string
	result := make([]rune, len(sampled))
	for i, v := range sampled {
		result[i] = valueToBlock(v, min, max)
	}

	style := lipgloss.NewStyle()
	if color != "" {
		style = style.Foreground(color)
	}

	return style.Render(string(result))
}

// SparklineWithThresholds renders a sparkline with color thresholds
func SparklineWithThresholds(values []float64, width int, warnThreshold, critThreshold float64, okColor, warnColor, critColor lipgloss.Color) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	sampled := sampleValues(values, width)

	min, max := sampled[0], sampled[0]
	for _, v := range sampled {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Build sparkline with per-character coloring
	var result string
	for _, v := range sampled {
		block := string(valueToBlock(v, min, max))

		var color lipgloss.Color
		if v >= critThreshold {
			color = critColor
		} else if v >= warnThreshold {
			color = warnColor
		} else {
			color = okColor
		}

		result += lipgloss.NewStyle().Foreground(color).Render(block)
	}

	return result
}

// sampleValues resamples the values slice to the target width
func sampleValues(values []float64, width int) []float64 {
	if len(values) == width {
		return values
	}

	result := make([]float64, width)

	if len(values) < width {
		// Pad with zeros at the beginning
		padding := width - len(values)
		for i := 0; i < padding; i++ {
			result[i] = 0
		}
		copy(result[padding:], values)
	} else {
		// Sample to fit
		ratio := float64(len(values)) / float64(width)
		for i := 0; i < width; i++ {
			idx := int(float64(i) * ratio)
			if idx >= len(values) {
				idx = len(values) - 1
			}
			result[i] = values[idx]
		}
	}

	return result
}

// valueToBlock converts a value to a block character based on its position in the range
func valueToBlock(value, min, max float64) rune {
	if max == min {
		return SparklineBlocks[len(SparklineBlocks)/2] // Middle block if all same
	}

	// Normalize to 0-1 range
	normalized := (value - min) / (max - min)

	// Map to block index
	idx := int(normalized * float64(len(SparklineBlocks)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(SparklineBlocks) {
		idx = len(SparklineBlocks) - 1
	}

	return SparklineBlocks[idx]
}
