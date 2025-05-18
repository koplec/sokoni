package tztime

import "time"

// Goのタイムゾーンは、IANAタイムゾーンに基づいているので、
// time.LoadLocation("Asia/Tokyo")を使うこともできるが、
// zoneinfoがOSにないことも考えて、明示的にoffsetを9*60*60secにしている。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

/**
 * Zone returns the time Zone for the application.
 * Currently, it is set to Japan Standard Time (JST).
 * This function is used to ensure that all time-related operations
 * are performed in the correct time Zone.
 *
 */
func Zone() *time.Location {
	return jst
}
