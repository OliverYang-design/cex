package cex

type Api struct {
	ApiKey     string `json:"apiKey,omitempty" bson:"apiKey"`
	SecretKey  string `json:"secretKey,omitempty" bson:"secretKey"`
	Passphrase string `json:"passphrase,omitempty" bson:"passphrase"`

	signer Signer
}

func (api Api) Sign(payload, key string) []byte {
	if api.signer != nil {
		return api.signer(payload, key)
	}
	return nil
}

type User interface {
	Cex() Cex
	Api() Api
}
