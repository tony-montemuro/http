package lws

const (
	SP = ' '
	HT = '\t'
	CR = '\r'
	LF = '\n'
)

func getMinAndMax(s string, i int) (int, int) {
	min := i + 1
	if s[i] == CR {
		min += 2
	}

	max := min
	for max < len(s) && (s[max] == SP || s[max] == HT) {
		max += 1
	}

	return min, max
}

func Check(s string, i int) (bool, int) {
	if i >= len(s) {
		return false, i
	}

	if s[i] != SP && s[i] != HT && s[i] != CR {
		return false, i
	}

	if s[i] == CR {
		if i+2 >= len(s) {
			return false, i
		}

		if s[i+1] != LF {
			return false, i
		}

		if s[i+2] != SP && s[i+2] != HT {
			return false, i
		}
	}

	_, max := getMinAndMax(s, i)
	return true, max
}

func NewLine(s string, i int) (bool, int, int) {
	if i+2 >= len(s) {
		return false, i, i
	}

	if s[i] != CR || s[i+1] != LF {
		return false, i, i
	}

	if s[i+2] != SP && s[i+2] != HT {
		return false, i, i
	}

	min, max := getMinAndMax(s, i)
	return true, min, max
}

func TrimLeft(s string) string {
	first := len(s)

	i := 0
	for i < first {
		isLws, next := Check(s, i)
		if !isLws {
			first = i
		}
		i = next
	}

	return s[first:]
}

func TrimRight(s string) string {
	last := -1
	i := 0

	for i < len(s) {
		isLws, next := Check(s, i)
		if !isLws {
			last = i
			i++
		} else {
			i = next
		}
	}

	return s[:last+1]
}

func Trim(s string) string {
	return TrimRight(TrimLeft(s))
}
