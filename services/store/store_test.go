package store

import (
	"fmt"
	"os"
	"testing"
	"strings"
	"github.com/ortutay/decloud/testutil"
)

func TestStoreBlob(t *testing.T) {
	defer os.RemoveAll(testutil.InitDir(t))
	r := strings.NewReader(strings.Repeat("abc", 1))
	blob, err := NewBlobFromReader(r)
	if err != nil {
		t.Fatal(err)
	}
	storeBlob(blob)
	blob2, err := NewBlobFromDisk(blob.ID)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("blob: %v\n", blob.ShortString())
	fmt.Printf("blob2: %v\n", blob2.ShortString())
	if blob.String() != blob2.String() {
		t.Fatalf("blobs do not match: %v != %v\n",
			blob.ShortString(), blob2.ShortString())
	}
	// TODO(ortutay): verify that files were written
	// TODO(ortutay): verify only 2 files were written
}
