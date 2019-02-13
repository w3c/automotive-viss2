package main


import "testing"



func TestUrlToPath(t *testing.T){
  path := urlToPath("/Root/Branch/Leaf")

  if path == "Root.Branch.Leaf"{
      t.Log("passed")
  } else {
      t.Fail()
  }
}

func TestGetSpeed(t *testing.T){
  speed := getSpeed()
  if (speed >= 0) && (speed <= 250){
    t.Log("Ok")
  } else {
    t.Fail()
  }

}
