package controllers

import "testing"

func TestPublicKey(t *testing.T){
	key, _ := PublicKey()
	t.Log(key)
}
