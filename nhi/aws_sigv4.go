package nhi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type awsCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
}

func buildAWSSTSAttestation(creds awsCredentials, region, runtime string, now time.Time) (string, error) {
	if strings.TrimSpace(creds.AccessKeyID) == "" || strings.TrimSpace(creds.SecretAccessKey) == "" {
		return "", fmt.Errorf("AWS credentials are missing access key or secret key")
	}
	if region = strings.TrimSpace(region); region == "" {
		region = "us-east-1"
	}
	host := "sts." + region + ".amazonaws.com"
	scopeDate := now.UTC().Format("20060102")
	amzDate := now.UTC().Format("20060102T150405Z")
	scope := scopeDate + "/" + region + "/sts/aws4_request"

	values := url.Values{}
	values.Set("Action", "GetCallerIdentity")
	values.Set("Version", "2011-06-15")
	values.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	values.Set("X-Amz-Credential", creds.AccessKeyID+"/"+scope)
	values.Set("X-Amz-Date", amzDate)
	values.Set("X-Amz-Expires", "60")
	values.Set("X-Amz-SignedHeaders", "host")
	if strings.TrimSpace(creds.Token) != "" {
		values.Set("X-Amz-Security-Token", creds.Token)
	}
	canonicalQuery := values.Encode()
	canonicalRequest := strings.Join([]string{
		"GET",
		"/",
		canonicalQuery,
		"host:" + host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")
	requestHash := sha256.Sum256([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hex.EncodeToString(requestHash[:]),
	}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(creds.SecretAccessKey, scopeDate, region), []byte(stringToSign)))
	values.Set("X-Amz-Signature", signature)

	envelope := map[string]string{
		"source":  "aws_sts",
		"runtime": runtime,
		"region":  region,
		"url":     "https://" + host + "/?" + values.Encode(),
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return "", fmt.Errorf("marshal AWS STS attestation envelope: %w", err)
	}
	return string(raw), nil
}

func signingKey(secret, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("sts"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
