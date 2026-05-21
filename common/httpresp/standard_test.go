package httpresp

import (
	"net/http"
	"testing"
)

func TestDefaultMsg(t *testing.T) {
	if DefaultMsg(http.StatusOK) != MsgOK {
		t.Fatal(http.StatusOK)
	}
	if DefaultMsg(http.StatusConflict) != MsgConflict {
		t.Fatal(http.StatusConflict)
	}
}
