package controllers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"log"
)

var privateKey *rsa.PrivateKey
var publicKey *rsa.PublicKey

func init(){
	var err error

	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalln(err.Error())
	}

	err = privateKey.Validate()
	if err != nil {
		log.Fatalln(err.Error())
	}

	publicKey = &privateKey.PublicKey
}

func RSADecrypt(ciphertext []byte)(out []byte, err error){
	out, err = rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	return
}

func PublicKey()(string, error){
	key, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func AESDecryptCFB(key, ciphertext []byte)([]byte, error){
	block, err := aes.NewCipher(key[aes.BlockSize:])
	if err != nil {
		return []byte(""), err
	}

	iv := make([]byte, aes.BlockSize)
	msg := make([]byte, len(ciphertext))


	copy(iv, key[:aes.BlockSize])
	copy(msg, ciphertext)

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(msg, msg)

	return msg, nil

}
