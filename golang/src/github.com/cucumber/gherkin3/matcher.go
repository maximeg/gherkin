package gherkin3

import (
	"regexp"
	"strings"
)

const (
	DEFAULT_DIALECT                 = "en"
	COMMENT_PREFIX                  = "#"
	TAG_PREFIX                      = "@"
	TITLE_KEYWORD_SEPARATOR         = ":"
	TABLE_CELL_SEPARATOR            = "|"
	DOCSTRING_SEPARATOR             = "\"\"\""
	DOCSTRING_ALTERNATIVE_SEPARATOR = "```"
)

type matcher struct {
	gdp                      GherkinDialectProvider
	lang                     string
	dialect                  *GherkinDialect
	activeDocStringSeparator string
	indentToRemove           int
	languagePattern          *regexp.Regexp
}

func NewMatcher(gdp GherkinDialectProvider) Matcher {
	return &matcher{
		gdp:             gdp,
		lang:            DEFAULT_DIALECT,
		dialect:         gdp.GetDialect(DEFAULT_DIALECT),
		languagePattern: regexp.MustCompile("^\\s*#\\s*language\\s*:\\s*([a-zA-Z\\-_]+)\\s*$"),
	}
}

func (m *matcher) newTokenAtLocation(line, index int) (token *Token) {
	column := index + 1
	token = new(Token)
	token.GherkinDialect = m.lang
	token.Location = &Location{line, column}
	return
}

// @Override
// public boolean match_EOF(Token token) {
//     if (token.isEOF()) {
//         SetTokenMatched(token, TokenType.EOF);
//         return true;
//     }
//     return false;
// }
func (m *matcher) MatchEOF(line *Line) (err error, ok bool, token *Token) {
	if line.IsEof() {
		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_EOF
	}
	return
}

// @Override
// public boolean match_Empty(Token token) {
//     if (token.line.isEmpty()) {
//         SetTokenMatched(token, TokenType.Empty);
//         return true;
//     }
//     return false;
// }
func (m *matcher) MatchEmpty(line *Line) (err error, ok bool, token *Token) {
	if line.IsEmpty() {
		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_Empty
	}
	return
}

// @Override
// public boolean match_Comment(Token token) {
//     if (token.line.startsWith(GherkinLanguageConstants.COMMENT_PREFIX)) {
//         String text = token.line.getLineText(0); //take the entire line
//         SetTokenMatched(token, TokenType.Comment, text, null, 0, null);
//         return true;
//     }
//     return false;
// }
func (m *matcher) MatchComment(line *Line) (err error, ok bool, token *Token) {
	if line.StartsWith(COMMENT_PREFIX) {
		token, ok = m.newTokenAtLocation(line.lineNumber, 0), true
		token.Type = TokenType_Comment
		token.Text = line.lineText
	}
	return
}

// @Override
// public boolean match_TagLine(Token token) {
//     if (token.line.startsWith(GherkinLanguageConstants.TAG_PREFIX)) {
//         SetTokenMatched(token, TokenType.TagLine, null, null, null, token.line.getTags());
//         return true;
//     }
//     return false;
// }
//
func (m *matcher) MatchTagLine(line *Line) (err error, ok bool, token *Token) {
	if line.StartsWith(TAG_PREFIX) {
		var tags []*LineSpan
		var column = line.Indent()
		splits := strings.Split(line.trimmedLineText, TAG_PREFIX)
		for i := range splits {
			txt := strings.Trim(splits[i], " ")
			if txt != "" {
				tags = append(tags, &LineSpan{column, TAG_PREFIX + txt})
			}
			column = column + len(splits[i]) + 1
		}

		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_TagLine
		token.Items = tags
	}
	return
}

// private boolean matchTitleLine(Token token, TokenType tokenType, List<String> keywords) {
//     for (String keyword : keywords) {
//         if (token.line.startsWithTitleKeyword(keyword)) {
//             String title = token.line.getRestTrimmed(keyword.length() + GherkinLanguageConstants.TITLE_KEYWORD_SEPARATOR.length());
//             SetTokenMatched(token, tokenType, title, keyword, null, null);
//             return true;
//         }
//     }
//     return false;
// }
func (m *matcher) matchTitleLine(line *Line, tokenType TokenType, keywords []string) (err error, ok bool, token *Token) {
	for i := range keywords {
		keyword := keywords[i]
		if line.StartsWith(keyword + TITLE_KEYWORD_SEPARATOR) {
			token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
			token.Type = tokenType
			token.Keyword = keyword
			token.Text = strings.Trim(line.trimmedLineText[len(keyword)+1:], " ")
			return
		}
	}
	return
}

func (m *matcher) MatchFeatureLine(line *Line) (err error, ok bool, token *Token) {
	return m.matchTitleLine(line, TokenType_FeatureLine, m.dialect.FeatureKeywords())
}
func (m *matcher) MatchBackgroundLine(line *Line) (err error, ok bool, token *Token) {
	return m.matchTitleLine(line, TokenType_BackgroundLine, m.dialect.BackgroundKeywords())
}
func (m *matcher) MatchScenarioLine(line *Line) (err error, ok bool, token *Token) {
	return m.matchTitleLine(line, TokenType_ScenarioLine, m.dialect.ScenarioKeywords())
}
func (m *matcher) MatchScenarioOutlineLine(line *Line) (err error, ok bool, token *Token) {
	return m.matchTitleLine(line, TokenType_ScenarioOutlineLine, m.dialect.ScenarioOutlineKeywords())
}
func (m *matcher) MatchExamplesLine(line *Line) (err error, ok bool, token *Token) {
	return m.matchTitleLine(line, TokenType_ExamplesLine, m.dialect.ExamplesKeywords())
}
func (m *matcher) MatchStepLine(line *Line) (err error, ok bool, token *Token) {
	keywords := m.dialect.StepKeywords()
	for i := range keywords {
		keyword := keywords[i]
		if line.StartsWith(keyword) {
			token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
			token.Type = TokenType_StepLine
			token.Keyword = keyword
			token.Text = strings.Trim(line.trimmedLineText[len(keyword):], " ")
			return
		}
	}
	return
}

// @Override
// public boolean match_DocStringSeparator(Token token) {
//     return activeDocStringSeparator == null
//             // open
//             ? match_DocStringSeparator(token, GherkinLanguageConstants.DOCSTRING_SEPARATOR, true) ||
//             match_DocStringSeparator(token, GherkinLanguageConstants.DOCSTRING_ALTERNATIVE_SEPARATOR, true)
//             // close
//             : match_DocStringSeparator(token, activeDocStringSeparator, false);
// }
//
func (m *matcher) MatchDocStringSeparator(line *Line) (err error, ok bool, token *Token) {
	if m.activeDocStringSeparator != "" {
		if line.StartsWith(m.activeDocStringSeparator) {
			// close
			token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
			token.Type = TokenType_DocStringSeparator

			m.indentToRemove = 0
			m.activeDocStringSeparator = ""
		}
		return
	}
	if line.StartsWith(DOCSTRING_SEPARATOR) {
		m.activeDocStringSeparator = DOCSTRING_SEPARATOR
	} else if line.StartsWith(DOCSTRING_ALTERNATIVE_SEPARATOR) {
		m.activeDocStringSeparator = DOCSTRING_ALTERNATIVE_SEPARATOR
	}
	if m.activeDocStringSeparator != "" {
		// open
		contentType := line.trimmedLineText[len(m.activeDocStringSeparator):]
		m.indentToRemove = line.Indent()
		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_DocStringSeparator
		token.Text = contentType
	}
	return
}

// @Override
// public boolean match_TableRow(Token token) {
//     if (token.line.startsWith(GherkinLanguageConstants.TABLE_CELL_SEPARATOR)) {
//         SetTokenMatched(token, TokenType.TableRow, null, null, null, token.line.getTableCells());
//         return true;
//     }
//     return false;
// }

func (m *matcher) MatchTableRow(line *Line) (err error, ok bool, token *Token) {
	if line.StartsWith(TABLE_CELL_SEPARATOR) {
		var cells []*LineSpan
		var column = line.Indent() + 1
		ttxt := strings.Trim(line.trimmedLineText, " ")
		splits := strings.Split(ttxt[1:len(ttxt)-1], TABLE_CELL_SEPARATOR)
		for i := range splits {
			ind := 0
			txt := splits[i]
			for k := range txt {
				if txt[k:k+1] != " " {
					break
				}
				ind++
			}
			cells = append(cells, &LineSpan{column + ind + 1, strings.Trim(splits[i], " ")})
			column = column + len(txt) + 1
		}

		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_TableRow
		token.Items = cells
	}
	return
}

// @Override
// public boolean match_Language(Token token) {
//     Matcher matcher = LANGUAGE_PATTERN.matcher(token.line.getLineText(0));
//     if (matcher.matches()) {
//         String language = matcher.group(1);
//         SetTokenMatched(token, TokenType.Language, language, null, null, null);
//
//         currentDialect = dialectProvider.getDialect(language, token.location);
//         return true;
//     }
//     return false;
// }
func (m *matcher) MatchLanguage(line *Line) (err error, ok bool, token *Token) {
	matches := m.languagePattern.FindStringSubmatch(line.trimmedLineText)
	if len(matches) > 0 {
		lang := matches[1]
		token, ok = m.newTokenAtLocation(line.lineNumber, line.Indent()), true
		token.Type = TokenType_Language
		token.Text = lang

		dialect := m.gdp.GetDialect(lang)
		if dialect == nil {
			err = &parseError{"Language not supported: " + lang, token.Location}
		} else {
			m.lang = lang
			m.dialect = dialect
		}
	}
	return
}

// @Override
// public boolean match_Other(Token token) {
//     String text = token.line.getLineText(indentToRemove); //take the entire line, except removing DocString indents
//     SetTokenMatched(token, TokenType.Other, text, null, 0, null);
//     return true;
// }
func (m *matcher) MatchOther(line *Line) (err error, ok bool, token *Token) {
	token, ok = m.newTokenAtLocation(line.lineNumber, 0), true
	token.Type = TokenType_Other

	txt := line.lineText
	var ind int
	for k := range txt {
		if txt[k:k+1] != " " {
			break
		}
		if ind >= m.indentToRemove {
			break
		}
		ind++
	}
	token.Text = txt[ind:]
	return
}