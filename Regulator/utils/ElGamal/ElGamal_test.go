package ElGamal

import (
	_ "math/big"
	"reflect"
	"testing"
)

func TestGenerateKeys(t *testing.T) {
	type args struct {
		info string
	}
	tests := []struct {
		name     string
		args     args
		wantPub  PublicKey
		wantPriv PrivateKey
		wantErr  bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPub, gotPriv, err := GenerateKeys(tt.args.info)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPub, tt.wantPub) {
				t.Errorf("GenerateKeys() gotPub = %v, want %v", gotPub, tt.wantPub)
			}
			if !reflect.DeepEqual(gotPriv, tt.wantPriv) {
				t.Errorf("GenerateKeys() gotPriv = %v, want %v", gotPriv, tt.wantPriv)
			}
		})
	}
}
