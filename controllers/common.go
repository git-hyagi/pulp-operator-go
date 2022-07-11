package controllers

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Generate a random string with length pwdSize
func createPwd(pwdSize int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	pwd := make([]byte, pwdSize)
	for i := range pwd {
		pwd[i] = chars[rand.Intn(len(chars))]
	}
	return string(pwd)
}

// Retrieve specific keys from secret object
func (r *PulpReconciler) retrieveSecretData(ctx context.Context, secretName, secretNamespace string, keys ...string) (map[string]string, error) {
	found := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, found)
	if err != nil {
		return nil, err
	}

	secret := map[string]string{}
	for _, key := range keys {
		secret[key] = string(found.Data[key])
	}

	return secret, nil
}

// https://github.com/golang/go/issues/20161#issuecomment-561560657
func SafeGo(log logr.Logger, f func()) {
	go func() {
		defer func() {
			if panicMessage := recover(); panicMessage != nil {
				err := fmt.Errorf("Panic: %v", panicMessage)
				log.Error(err, err.Error())
			}
		}()

		f()
	}()
}
