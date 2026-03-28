package ui

import "strings"

// normalizeHotkey maps Russian JCUKEN symbols to the same physical positions
// on a US QWERTY keyboard so command hotkeys keep working across layouts.
func normalizeHotkey(key string) string {
	if key == "" {
		return key
	}

	prefix := ""
	base := key
	if idx := strings.LastIndex(key, "+"); idx >= 0 {
		prefix = key[:idx+1]
		base = key[idx+1:]
	}

	if alias, ok := hotkeyAliases[base]; ok {
		return prefix + alias
	}

	return key
}

var hotkeyAliases = map[string]string{
	"\u0451": "`",
	"\u0439": "q",
	"\u0446": "w",
	"\u0443": "e",
	"\u043a": "r",
	"\u0435": "t",
	"\u043d": "y",
	"\u0433": "u",
	"\u0448": "i",
	"\u0449": "o",
	"\u0437": "p",
	"\u0445": "[",
	"\u044a": "]",
	"\u0444": "a",
	"\u044b": "s",
	"\u0432": "d",
	"\u0430": "f",
	"\u043f": "g",
	"\u0440": "h",
	"\u043e": "j",
	"\u043b": "k",
	"\u0434": "l",
	"\u0436": ";",
	"\u044d": "'",
	"\u044f": "z",
	"\u0447": "x",
	"\u0441": "c",
	"\u043c": "v",
	"\u0438": "b",
	"\u0442": "n",
	"\u044c": "m",
	"\u0431": ",",
	"\u044e": ".",
	"\u0401": "~",
	"\u0419": "Q",
	"\u0426": "W",
	"\u0423": "E",
	"\u041a": "R",
	"\u0415": "T",
	"\u041d": "Y",
	"\u0413": "U",
	"\u0428": "I",
	"\u0429": "O",
	"\u0417": "P",
	"\u0425": "{",
	"\u042a": "}",
	"\u0424": "A",
	"\u042b": "S",
	"\u0412": "D",
	"\u0410": "F",
	"\u041f": "G",
	"\u0420": "H",
	"\u041e": "J",
	"\u041b": "K",
	"\u0414": "L",
	"\u0416": ":",
	"\u042d": "\"",
	"\u042f": "Z",
	"\u0427": "X",
	"\u0421": "C",
	"\u041c": "V",
	"\u0418": "B",
	"\u0422": "N",
	"\u042c": "M",
	"\u0411": "<",
	"\u042e": ">",
}
