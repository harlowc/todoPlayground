package main

import (
	"regexp"
	"strings"
	"time"
)

var (
	quickAddCategoryToken = regexp.MustCompile(`^@([A-Za-z0-9][A-Za-z0-9_-]*)$`)
	quickAddPriorityToken = regexp.MustCompile(`(?i)^p([123])$`)
	quickAddISODateToken  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

type quickAddToken struct {
	raw   string
	clean string
}

func applyQuickAdd(input todoInput, today time.Time) todoInput {
	tokens := quickAddTokens(input.Text)
	if len(tokens) == 0 {
		input.Text = ""
		return input
	}

	kept := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		token := tokens[i]

		if dueDate, consumed, ok := quickAddDateAt(tokens, i, today); ok {
			if input.DueDate == "" {
				input.DueDate = dueDate
			}
			i += consumed - 1
			continue
		}

		if category, consumed, ok := quickAddCategoryAt(tokens, i); ok {
			if input.Category == "" {
				input.Category = category
			}
			i += consumed - 1
			continue
		}

		if matches := quickAddCategoryToken.FindStringSubmatch(token.clean); matches != nil {
			if input.Category == "" {
				input.Category = matches[1]
			}
			continue
		}

		if priority, consumed, ok := quickAddPriorityAt(tokens, i); ok {
			if input.Priority == "" || input.Priority == "normal" {
				input.Priority = priority
			}
			i += consumed - 1
			continue
		}

		kept = append(kept, token.raw)
	}

	input.Text = cleanQuickAddText(strings.Join(kept, " "))
	return input
}

func quickAddTokens(text string) []quickAddToken {
	words := strings.Fields(text)
	tokens := make([]quickAddToken, 0, len(words))
	for _, word := range words {
		clean := cleanQuickAddToken(word)
		if clean == "" {
			continue
		}
		tokens = append(tokens, quickAddToken{raw: word, clean: clean})
	}
	return tokens
}

func cleanQuickAddToken(token string) string {
	return strings.Trim(token, " \t\r\n.,;:!?()[]{}\"'")
}

func cleanQuickAddText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, " ,;:")
	return strings.Join(strings.Fields(text), " ")
}

func quickAddDateAt(tokens []quickAddToken, index int, today time.Time) (string, int, bool) {
	token := tokens[index]
	switch {
	case quickAddDateCue(token.clean):
		if index+1 >= len(tokens) {
			return "", 0, false
		}
		dueDate, consumed, ok := quickAddDateValueAt(tokens, index+1, today)
		if !ok {
			return "", 0, false
		}
		return dueDate, consumed + 1, true
	default:
		if index == 0 {
			return "", 0, false
		}
		return quickAddDateValueAt(tokens, index, today)
	}
}

func quickAddDateCue(word string) bool {
	switch strings.ToLower(word) {
	case "by", "due", "on":
		return true
	default:
		return false
	}
}

func quickAddDateValueAt(tokens []quickAddToken, index int, today time.Time) (string, int, bool) {
	token := tokens[index]
	switch {
	case strings.EqualFold(token.clean, "today"):
		return today.Format("2006-01-02"), 1, true
	case strings.EqualFold(token.clean, "tomorrow"):
		return tomorrow(today), 1, true
	case quickAddISODateToken.MatchString(token.clean):
		return token.clean, 1, true
	case strings.EqualFold(token.clean, "next") && index+1 < len(tokens) && strings.EqualFold(tokens[index+1].clean, "week"):
		return today.AddDate(0, 0, 7).Format("2006-01-02"), 2, true
	default:
		return "", 0, false
	}
}

func quickAddCategoryAt(tokens []quickAddToken, index int) (string, int, bool) {
	if !quickAddCategoryCue(tokens[index].clean) || index+1 >= len(tokens) {
		return "", 0, false
	}

	category := make([]string, 0, 2)
	for i := index + 1; i < len(tokens); i++ {
		if quickAddCategoryBoundary(tokens, i) {
			break
		}
		category = append(category, tokens[i].clean)
		if strings.HasSuffix(tokens[i].raw, ",") || strings.HasSuffix(tokens[i].raw, ";") {
			i++
			return cleanQuickAddText(strings.Join(category, " ")), i - index, len(category) > 0
		}
	}
	if len(category) == 0 {
		return "", 0, false
	}
	return cleanQuickAddText(strings.Join(category, " ")), len(category) + 1, true
}

func quickAddCategoryCue(word string) bool {
	switch strings.ToLower(word) {
	case "category", "label":
		return true
	default:
		return false
	}
}

func quickAddCategoryBoundary(tokens []quickAddToken, index int) bool {
	token := tokens[index]
	if quickAddCategoryCue(token.clean) {
		return true
	}
	if _, _, ok := quickAddDateAt(tokens, index, time.Time{}); ok {
		return true
	}
	if _, _, ok := quickAddPriorityAt(tokens, index); ok {
		return true
	}
	if quickAddCategoryToken.MatchString(token.clean) {
		return true
	}
	return false
}

func quickAddPriorityAt(tokens []quickAddToken, index int) (string, int, bool) {
	token := tokens[index]
	if matches := quickAddPriorityToken.FindStringSubmatch(token.clean); matches != nil {
		return quickAddPriority(matches[1]), 1, true
	}
	if priority, ok := quickAddPriorityWord(token.clean); ok && index+1 < len(tokens) && strings.EqualFold(tokens[index+1].clean, "priority") {
		return priority, 2, true
	}
	if strings.EqualFold(token.clean, "priority") && index+1 < len(tokens) {
		if priority, ok := quickAddPriorityWord(tokens[index+1].clean); ok {
			return priority, 2, true
		}
	}
	return "", 0, false
}

func quickAddPriorityWord(word string) (string, bool) {
	switch strings.ToLower(word) {
	case "high":
		return "high", true
	case "normal", "medium":
		return "normal", true
	case "low":
		return "low", true
	default:
		return "", false
	}
}

func quickAddPriority(level string) string {
	switch level {
	case "1":
		return "high"
	case "2":
		return "normal"
	case "3":
		return "low"
	default:
		return "normal"
	}
}
