package main


import "testing"


func TestUrlToPath(t *testing.T){
  path = urlToPath("/Root/Branch/Leaf")
  if path != "Root.Branch.Leaf" {
      t.Fail()
  }

}
