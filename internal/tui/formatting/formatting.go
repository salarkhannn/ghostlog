package formatting

import "fmt"

func Bytes(n int) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1fmb", float64(n)/1024/1024)
	case n >= 1024:
		return fmt.Sprintf("%.1fkb", float64(n)/1024)
	default:
		return fmt.Sprintf("%db", n)
	}
}

func Duration(secs int64) string {
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm%ds", secs/60, secs%60)
	default:
		return fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
	}
}
