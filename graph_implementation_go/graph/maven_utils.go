package graph

import "regexp"

var reg = regexp.MustCompile("((?P<open>[\\(\\[])(?P<bothVer>((?P<firstVer>(0|[1-9]+)(\\.(0|[1-9]+)(\\.(0|[1-9]+))?)?)(?P<comma1>,)(?P<secondVer1>(0|[1-9]+)(\\.(0|[1-9]+)(\\.(0|[1-9]+))?)?)?)|((?P<comma2>,)?(?P<secondVer2>(0|[1-9]+)(\\.(0|[1-9]+)(\\.(0|[1-9]+))?)?)?))(?P<close>[\\)\\]]))|(?P<simplevers>(0|[1-9]+)(\\.(0|[1-9]+)(\\.(0|[1-9]+))?)?)")

func ParseMultipleMavenSemanticVersions(s string) string {
	var finalResult string
	chars := []rune(s)
	openIndex := 0
	closeIndex := 0
	for i := 0; i < len(chars); i++ {
		char := string(chars[i])
		if char == "(" || char == "[" {
			openIndex = i
		}
		if char == ")" || char == "]" {
			closeIndex = i
			if i != len(chars)-1 {
				finalResult += translateMavenSemver(s[openIndex:closeIndex+1]) + " || "
			} else {
				finalResult += translateMavenSemver(s[openIndex : closeIndex+1])
			}
		}

	}
	if closeIndex == 0 && openIndex == 0 {
		return translateMavenSemver(s)
	}

	return finalResult
}

func translateMavenSemver(s string) string {
	match := reg.FindStringSubmatch(s)
	result := make(map[string]string)
	var finalResult string
	for i, name := range reg.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	if len(result["close"]) > 0 {
		if len(result["secondVer2"]) > 0 {
			if len(result["comma1"]) > 0 || len(result["comma2"]) > 0 {
				switch result["close"] {
				case "]":
					finalResult = "<= " + result["secondVer2"]
				case ")":
					finalResult = "< " + result["secondVer2"]
				}
			} else {
				finalResult = "= " + result["secondVer2"]
			}
		} else {
			if len(result["firstVer"]) > 0 && len(result["secondVer1"]) > 0 {
				switch result["open"] {
				case "[":
					finalResult = ">= " + result["firstVer"] + ", "
				case "(":
					finalResult = "> " + result["firstVer"] + ", "
				}
				switch result["close"] {
				case "]":
					finalResult += "<= " + result["secondVer1"]
				case ")":
					finalResult += "< " + result["secondVer1"]
				}
			} else if len(result["firstVer"]) > 0 && len(result["secondVer1"]) == 0 {
				switch result["open"] {
				case "[":
					finalResult = ">= " + result["firstVer"]
				case "(":
					finalResult = "> " + result["firstVer"]
				}
			}
		}
	} else {
		finalResult = ">= " + result["simplevers"]
	}
	return finalResult

}
