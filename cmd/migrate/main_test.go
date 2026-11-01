package main

import "testing"

func TestSplitSQLDelimiter(t *testing.T) {
	in := "CREATE TABLE x(id int);\nDELIMITER $$\nCREATE TRIGGER t BEFORE INSERT ON x FOR EACH ROW BEGIN SET @x=1; END$$\nDELIMITER ;\nINSERT INTO x VALUES(1);"
	got := splitSQL(in)
	if len(got) != 3 {
		t.Fatalf("got %d statements: %#v", len(got), got)
	}
}
