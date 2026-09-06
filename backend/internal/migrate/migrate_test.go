package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSortsAndHashesSQLOnly(t *testing.T){
	dir:=t.TempDir()
	if err:=os.WriteFile(filepath.Join(dir,"000002_b.sql"),[]byte("SELECT 2;"),0600); err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(filepath.Join(dir,"000001_a.sql"),[]byte("SELECT 1;"),0600); err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(filepath.Join(dir,"notes.txt"),[]byte("ignore"),0600); err!=nil{t.Fatal(err)}
	items,err:=discover(dir); if err!=nil{t.Fatal(err)}
	if len(items)!=2 || items[0].Name!="000001_a.sql" || items[1].Name!="000002_b.sql"{t.Fatalf("unexpected migration order: %#v",items)}
	if items[0].Checksum==items[1].Checksum{t.Fatal("different SQL produced same checksum")}
}
