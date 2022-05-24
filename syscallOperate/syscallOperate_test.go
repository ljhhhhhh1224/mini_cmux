package syscallOperate

import (
	"syscall"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCloseProcess(t *testing.T) {
	c := GetSyscallChan()
	Convey("Test channel", t, func() {
		c <- syscall.SIGTERM
		So(<-c, ShouldEqual, syscall.SIGTERM)
	})
}
