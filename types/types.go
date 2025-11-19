package types

type S3Credentials struct {
	Bucket    string `json:"bucket,omitempty" cbor:"b"`
	Url       string `json:"url,omitempty" cbor:"u"`
	AccessKey string `json:"access_key,omitempty" cbor:"a"`
	SecretKey string `json:"secret_key,omitempty" cbor:"s"`
	Region    string `json:"region,omitempty" cbor:"r"`
}
