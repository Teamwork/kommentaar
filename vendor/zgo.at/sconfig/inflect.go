// This is bitbucket.org/pkg/inflect stripped down to the parts that we need
// (Camelize and Pluralize) and modified somewhat to fit better with the sconfig
// API.
//
// Copyright © 2011 Chris Farmiloe
// Copyright © 2017 Martin Tournoij
// See the bottom of this file for the full copyright.
//
// TODO: Add tests, too
// TODO: Probably want to export some of this, so users can add their own.

package sconfig

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// inflectSet is the configuration for the pluralization rules. You can extend
// the rules with the Add* methods.
type inflectSet struct {
	uncountables map[string]bool
	plurals      []*rule
	singulars    []*rule
	humans       []*rule
	//acronyms       []*rule
	//acronymMatcher *regexp.Regexp
}

type rule struct {
	suffix      string
	replacement string
	exact       bool
}

var inflect *inflectSet

func init() {
	inflect = newRuleset()
	setDefault(inflect)
}

// NewInflectSet create a blank InflectSet. Unless you are going to build your
// own rules from scratch you probably won't need this and can just use the
// defaultRuleset via the global inflect.* methods.
func newRuleset() *inflectSet {
	rs := new(inflectSet)
	rs.uncountables = make(map[string]bool)
	rs.plurals = make([]*rule, 0)
	rs.singulars = make([]*rule, 0)
	rs.humans = make([]*rule, 0)
	//rs.acronyms = make([]*rule, 0)
	return rs
}

// setDefault sets the default set of common English pluralization rules for an
// inflectSet.
func setDefault(rs *inflectSet) {
	rs.AddPlural("s", "s")
	rs.AddPlural("testis", "testes")
	rs.AddPlural("axis", "axes")
	rs.AddPlural("octopus", "octopi")
	rs.AddPlural("virus", "viri")
	rs.AddPlural("octopi", "octopi")
	rs.AddPlural("viri", "viri")
	rs.AddPlural("alias", "aliases")
	rs.AddPlural("status", "statuses")
	rs.AddPlural("bus", "buses")
	rs.AddPlural("buffalo", "buffaloes")
	rs.AddPlural("tomato", "tomatoes")
	rs.AddPlural("tum", "ta")
	rs.AddPlural("ium", "ia")
	rs.AddPlural("ta", "ta")
	rs.AddPlural("ia", "ia")
	rs.AddPlural("sis", "ses")
	rs.AddPlural("lf", "lves")
	rs.AddPlural("rf", "rves")
	rs.AddPlural("afe", "aves")
	rs.AddPlural("bfe", "bves")
	rs.AddPlural("cfe", "cves")
	rs.AddPlural("dfe", "dves")
	rs.AddPlural("efe", "eves")
	rs.AddPlural("gfe", "gves")
	rs.AddPlural("hfe", "hves")
	rs.AddPlural("ife", "ives")
	rs.AddPlural("jfe", "jves")
	rs.AddPlural("kfe", "kves")
	rs.AddPlural("lfe", "lves")
	rs.AddPlural("mfe", "mves")
	rs.AddPlural("nfe", "nves")
	rs.AddPlural("ofe", "oves")
	rs.AddPlural("pfe", "pves")
	rs.AddPlural("qfe", "qves")
	rs.AddPlural("rfe", "rves")
	rs.AddPlural("sfe", "sves")
	rs.AddPlural("tfe", "tves")
	rs.AddPlural("ufe", "uves")
	rs.AddPlural("vfe", "vves")
	rs.AddPlural("wfe", "wves")
	rs.AddPlural("xfe", "xves")
	rs.AddPlural("yfe", "yves")
	rs.AddPlural("zfe", "zves")
	rs.AddPlural("hive", "hives")
	rs.AddPlural("quy", "quies")
	rs.AddPlural("by", "bies")
	rs.AddPlural("cy", "cies")
	rs.AddPlural("dy", "dies")
	rs.AddPlural("fy", "fies")
	rs.AddPlural("gy", "gies")
	rs.AddPlural("hy", "hies")
	rs.AddPlural("jy", "jies")
	rs.AddPlural("ky", "kies")
	rs.AddPlural("ly", "lies")
	rs.AddPlural("my", "mies")
	rs.AddPlural("ny", "nies")
	rs.AddPlural("py", "pies")
	rs.AddPlural("qy", "qies")
	rs.AddPlural("ry", "ries")
	rs.AddPlural("sy", "sies")
	rs.AddPlural("ty", "ties")
	rs.AddPlural("vy", "vies")
	rs.AddPlural("wy", "wies")
	rs.AddPlural("xy", "xies")
	rs.AddPlural("zy", "zies")
	rs.AddPlural("x", "xes")
	rs.AddPlural("ch", "ches")
	rs.AddPlural("ss", "sses")
	rs.AddPlural("sh", "shes")
	rs.AddPlural("matrix", "matrices")
	rs.AddPlural("vertix", "vertices")
	rs.AddPlural("indix", "indices")
	rs.AddPlural("matrex", "matrices")
	rs.AddPlural("vertex", "vertices")
	rs.AddPlural("index", "indices")
	rs.AddPlural("mouse", "mice")
	rs.AddPlural("louse", "lice")
	rs.AddPlural("mice", "mice")
	rs.AddPlural("lice", "lice")
	rs.AddPluralExact("ox", "oxen", true)
	rs.AddPluralExact("oxen", "oxen", true)
	rs.AddPluralExact("quiz", "quizzes", true)
	rs.AddSingular("s", "")
	rs.AddSingular("news", "news")
	rs.AddSingular("ta", "tum")
	rs.AddSingular("ia", "ium")
	rs.AddSingular("analyses", "analysis")
	rs.AddSingular("bases", "basis")
	rs.AddSingular("diagnoses", "diagnosis")
	rs.AddSingular("parentheses", "parenthesis")
	rs.AddSingular("prognoses", "prognosis")
	rs.AddSingular("synopses", "synopsis")
	rs.AddSingular("theses", "thesis")
	rs.AddSingular("analyses", "analysis")
	rs.AddSingular("aves", "afe")
	rs.AddSingular("bves", "bfe")
	rs.AddSingular("cves", "cfe")
	rs.AddSingular("dves", "dfe")
	rs.AddSingular("eves", "efe")
	rs.AddSingular("gves", "gfe")
	rs.AddSingular("hves", "hfe")
	rs.AddSingular("ives", "ife")
	rs.AddSingular("jves", "jfe")
	rs.AddSingular("kves", "kfe")
	rs.AddSingular("lves", "lfe")
	rs.AddSingular("mves", "mfe")
	rs.AddSingular("nves", "nfe")
	rs.AddSingular("oves", "ofe")
	rs.AddSingular("pves", "pfe")
	rs.AddSingular("qves", "qfe")
	rs.AddSingular("rves", "rfe")
	rs.AddSingular("sves", "sfe")
	rs.AddSingular("tves", "tfe")
	rs.AddSingular("uves", "ufe")
	rs.AddSingular("vves", "vfe")
	rs.AddSingular("wves", "wfe")
	rs.AddSingular("xves", "xfe")
	rs.AddSingular("yves", "yfe")
	rs.AddSingular("zves", "zfe")
	rs.AddSingular("hives", "hive")
	rs.AddSingular("tives", "tive")
	rs.AddSingular("lves", "lf")
	rs.AddSingular("rves", "rf")
	rs.AddSingular("quies", "quy")
	rs.AddSingular("bies", "by")
	rs.AddSingular("cies", "cy")
	rs.AddSingular("dies", "dy")
	rs.AddSingular("fies", "fy")
	rs.AddSingular("gies", "gy")
	rs.AddSingular("hies", "hy")
	rs.AddSingular("jies", "jy")
	rs.AddSingular("kies", "ky")
	rs.AddSingular("lies", "ly")
	rs.AddSingular("mies", "my")
	rs.AddSingular("nies", "ny")
	rs.AddSingular("pies", "py")
	rs.AddSingular("qies", "qy")
	rs.AddSingular("ries", "ry")
	rs.AddSingular("sies", "sy")
	rs.AddSingular("ties", "ty")
	rs.AddSingular("vies", "vy")
	rs.AddSingular("wies", "wy")
	rs.AddSingular("xies", "xy")
	rs.AddSingular("zies", "zy")
	rs.AddSingular("series", "series")
	rs.AddSingular("movies", "movie")
	rs.AddSingular("xes", "x")
	rs.AddSingular("ches", "ch")
	rs.AddSingular("sses", "ss")
	rs.AddSingular("shes", "sh")
	rs.AddSingular("mice", "mouse")
	rs.AddSingular("lice", "louse")
	rs.AddSingular("buses", "bus")
	rs.AddSingular("oes", "o")
	rs.AddSingular("shoes", "shoe")
	rs.AddSingular("crises", "crisis")
	rs.AddSingular("axes", "axis")
	rs.AddSingular("testes", "testis")
	rs.AddSingular("octopi", "octopus")
	rs.AddSingular("viri", "virus")
	rs.AddSingular("statuses", "status")
	rs.AddSingular("aliases", "alias")
	rs.AddSingularExact("oxen", "ox", true)
	rs.AddSingular("vertices", "vertex")
	rs.AddSingular("indices", "index")
	rs.AddSingular("matrices", "matrix")
	rs.AddSingularExact("quizzes", "quiz", true)
	rs.AddSingular("databases", "database")
	rs.AddIrregular("person", "people")
	rs.AddIrregular("man", "men")
	rs.AddIrregular("child", "children")
	rs.AddIrregular("sex", "sexes")
	rs.AddIrregular("move", "moves")
	rs.AddIrregular("zombie", "zombies")
	rs.AddUncountable("equipment")
	rs.AddUncountable("information")
	rs.AddUncountable("rice")
	rs.AddUncountable("money")
	rs.AddUncountable("species")
	rs.AddUncountable("series")
	rs.AddUncountable("fish")
	rs.AddUncountable("sheep")
	rs.AddUncountable("jeans")
	rs.AddUncountable("police")
}

// "dino_party" -> "DinoParty"
func (rs *inflectSet) camelize(word string) string {
	words := splitAtCaseChangeWithTitlecase(word)
	return strings.Join(words, "")
}

// returns the plural form of a singular word
func (rs *inflectSet) pluralize(word string) string {
	if len(word) == 0 {
		return word
	}
	if rs.isUncountable(word) {
		return word
	}
	for _, rule := range rs.plurals {
		if rule.exact {
			if word == rule.suffix {
				return rule.replacement
			}
		} else {
			if strings.HasSuffix(word, rule.suffix) {
				return replaceLast(word, rule.suffix, rule.replacement)
			}
		}
	}
	return word + "s"
}

// returns the singular form of a plural word
func (rs *inflectSet) singularize(word string) string {
	if len(word) == 0 {
		return word
	}
	if rs.isUncountable(word) {
		return word
	}
	for _, rule := range rs.singulars {
		if rule.exact {
			if word == rule.suffix {
				return rule.replacement
			}
		} else {
			if strings.HasSuffix(word, rule.suffix) {
				return replaceLast(word, rule.suffix, rule.replacement)
			}
		}
	}
	return word
}

// togglePlural will return the plural if word is singular, or the singular if
// the word is plural.
func (rs *inflectSet) togglePlural(word string) string {
	toggle := rs.pluralize(word)
	if toggle == word {
		toggle = rs.singularize(word)
	}

	return toggle
}

// Add a pluralization rule.
func (rs *inflectSet) AddPlural(suffix, replacement string) {
	rs.AddPluralExact(suffix, replacement, false)
}

// Add a pluralization rule with full string match.
func (rs *inflectSet) AddPluralExact(suffix, replacement string, exact bool) {
	// remove uncountable
	delete(rs.uncountables, suffix)
	// create rule
	r := new(rule)
	r.suffix = suffix
	r.replacement = replacement
	r.exact = exact
	// prepend
	rs.plurals = append([]*rule{r}, rs.plurals...)
}

// Add a singular rule.
func (rs *inflectSet) AddSingular(suffix, replacement string) {
	rs.AddSingularExact(suffix, replacement, false)
}

// same as AddSingular but you can set `exact` to force
// a full string match
func (rs *inflectSet) AddSingularExact(suffix, replacement string, exact bool) {
	// remove from uncountable
	delete(rs.uncountables, suffix)
	// create rule
	r := new(rule)
	r.suffix = suffix
	r.replacement = replacement
	r.exact = exact
	rs.singulars = append([]*rule{r}, rs.singulars...)
}

// Add any inconsistent pluralizing/sinularizing rules to the set here.
func (rs *inflectSet) AddIrregular(singular, plural string) {
	delete(rs.uncountables, singular)
	delete(rs.uncountables, plural)
	rs.AddPlural(singular, plural)
	rs.AddPlural(plural, plural)
	rs.AddSingular(plural, singular)
}

// add a word to this inflectRuleset that has the same singular and plural form
// for example: "rice"
func (rs *inflectSet) AddUncountable(word string) {
	rs.uncountables[strings.ToLower(word)] = true
}

func (rs *inflectSet) isUncountable(word string) bool {
	// handle multiple words by using the last one
	words := strings.Split(word, " ")
	if _, exists := rs.uncountables[strings.ToLower(words[len(words)-1])]; exists {
		return true
	}
	return false
}

func splitAtCaseChangeWithTitlecase(s string) []string {
	words := []string{}
	word := []rune{}

	for _, c := range s {
		spacer := isSpacerChar(c)
		if len(word) > 0 {
			if unicode.IsUpper(c) || spacer {
				words = append(words, string(word))
				word = make([]rune, 0)
			}
		}
		if !spacer {
			if len(word) > 0 {
				word = append(word, unicode.ToLower(c))
			} else {
				word = append(word, unicode.ToUpper(c))
			}
		}
	}
	words = append(words, string(word))
	return words
}

func replaceLast(s, match, repl string) string {
	// reverse strings
	srev := reverse(s)
	mrev := reverse(match)
	rrev := reverse(repl)
	// match first and reverse back
	return reverse(strings.Replace(srev, mrev, rrev, 1))
}

func isSpacerChar(c rune) bool {
	switch {
	case c == rune("_"[0]):
		return true
	case c == rune(" "[0]):
		return true
	case c == rune(":"[0]):
		return true
	case c == rune("-"[0]):
		return true
	}
	return false
}

func reverse(s string) string {
	o := make([]rune, utf8.RuneCountInString(s))
	i := len(o)
	for _, c := range s {
		i--
		o[i] = c
	}
	return string(o)
}

// The MIT License (MIT)
//
// Copyright © 2011 Chris Farmiloe
// Copyright © 2016-2017 Martin Tournoij
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// The software is provided "as is", without warranty of any kind, express or
// implied, including but not limited to the warranties of merchantability,
// fitness for a particular purpose and noninfringement. In no event shall the
// authors or copyright holders be liable for any claim, damages or other
// liability, whether in an action of contract, tort or otherwise, arising
// from, out of or in connection with the software or the use or other dealings
// in the software.
