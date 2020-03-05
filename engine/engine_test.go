package engine

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	ssqs "github.com/ONSdigital/dp-ssqs"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/sqsiface"
)

type handlerError struct {
	Field1, Field2 string
}

func (e *handlerError) Error() string {
	return "handler error"
}

var publicKey = `
-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBFqNbbgBCADdP1SMyrXWzjGtDDVHhd8yVlNVcd7XG6Me+C6YfbC1jhNJmGkN
BYIaSnIlE/a0PyFBuD+vyl50T0cMveIOvYj4A4SBfn0IK1jHGKUWqLv3Z7MBSCnF
9UDduw48Ncgf6YP8e+wJKtU8s8f2L2OhBgTCmbFarWIYfJewF6JIGSkx1OddjIz+
P4oKI9C943HGMEST7+bTfQtAxMon/XlHE1AdqwLnNPoKfcN1//MTS1l7ky5MWPdI
Lx+aF6phrv4RjJxXhvFv+PYWgKgPIjRls3/PX/w9KQ0WyCBqkUOsrKtDA49zrDTk
g3tnbMOq1dbqBShZNhJJE+JRhhEPTyD5qigJABEBAAG0NG9ucy1jb25jb3Vyc2Ut
c3FzIDxkZXYuZGlnaXRhbHB1Ymxpc2hpbmdAb25zLmdvdi51az6JAVQEEwEIAD4W
IQT+aIBQlEdvmXPBAbPiPUwJsHXcTAUCWo1tuAIbAwUJA8JnAAULCQgHAgYVCgkI
CwIEFgIDAQIeAQIXgAAKCRDiPUwJsHXcTFDEB/4x2Af4ETT6FJSraN5iLC5lJW8E
u95WRAQ4Uuuxh/Bm459ugXjfiMFWuJXqdFUHlpXVf6rN5NG5c9MBn+QDDTSutHRR
ojeY0YKUKOq4PkQ+68Vjkhxmq+RP8jQSZaMjIcECdpf9eVNTKy28YGK3Ku5E1+h0
5pVen797pJJE2H9FdRiN8dCxdUugGhG62RYB6AdbCEgNOwmz0S4GdqjJbKqUNK/6
MHVFIGz7lp22AsNfIz530pJrOuFkXwm4zfBMySxbs+Il3xoho4TPcFg0nSHqYckw
XhavDpoglpPb67NY3NF/MxHpJelvCeOne/3EPOumA8KNvZ2LAO/SZexLdtDeuQEN
BFqNbbgBCADEL44RGeNHq1ZI1RRfl/Ee3XYH3mm8cBFvXKgh3eEeb5JsqO8XlYJx
MS0wSeSWKzRAAh7XG6e+R/DJZ3w0lOCAKp6Xwd4etfC/8u8c9YH/sUi1Z2rR1i4k
PAX6cYDy+Hs8d2vgL0HUx6CIe1MET00ZzgXdmYpXpPmwlgB0Solc9LkQL7DwUKu0
HY6qBXF4+Q9JH6X9oDHy/qSqwAcC7DhKNU1FmZ5AJVplocvjlLFE3Nl77I6i0EwH
ySMEx2x1xytCHBqh2gQoo8QEgMdyt6nspalQHEjOYhLuWVdNc91NhuLz9Bft/KHz
z+4vvUaG+NtodBJCEZuJXdyvyk7ZdOHLABEBAAGJATwEGAEIACYWIQT+aIBQlEdv
mXPBAbPiPUwJsHXcTAUCWo1tuAIbDAUJA8JnAAAKCRDiPUwJsHXcTH/vB/wPlP4a
3dBP8+z8ojKzW9oyg+xq9hlxG1tVc/vohlwRZ1rYyQkxeabNrReD2VoQTl+KdQbe
C38mYFWD/8LmCXojXvRGhEHhKcUcPiRZ2Ir+bb2f4SWmoArhkqX9GsVW5XipOwfc
qqcGW+NyrBJ69CwZQ8CEFme0uk7Df4mYz3nItsCRMv1Bqp2M6u7ehTjN2e+zoo7J
BV+XMiVDF/wL+WiZTsR7HLOWKFRP4WTQKokqmOVbmQGjymt3yECkvXMHh1uIVg7G
QVdB8p4vOfeHCQY02BU5q5AGl6Z/vhTuCMFXA3ezepmWL9mPX07KEKsdPlC3DPyn
QfQuktF4OUZkBY9I
=oYL/
-----END PGP PUBLIC KEY BLOCK-----`

var emptyMessageBody = `
-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256


-----BEGIN PGP SIGNATURE-----

iQFVBAEBCAA/FiEE/miAUJRHb5lzwQGz4j1MCbB13EwFAlqVdEwhHGRldi5kaWdp
dGFscHVibGlzaGluZ0BvbnMuZ292LnVrAAoJEOI9TAmwddxM30IH/jIBaV59TQUd
eZBuZ5EolHURu8jet0hNdrSNbMqTNz1fi+GeWjd3vxm9OmWPV5RyL946uSqpnHuc
oowFWfYrX6mBOUMxWK7753/dT3zfJMlAs2M/edZkVc9P5ZfLCnDWADvqk/tutHxP
Xs3ACnDqMis5VOD/mYRAHS5Vk+Sp7+BcQZI9FLfbnQ6G+Mq920Ddm4HPjt58jBL8
3T4e8F1D9zcp9LEAhM32nKF7hXjt3v6QFtkkPdZuxW1nnlPfrw9KogCAwB1HcN5v
+t14lGRV6yt7t3X9vXHVhij34I0kgUQzRR3C/8TcfTWQYgvYRmdg7oqA2ooJpmMp
0ZIRlybX+1c=
=5+qW
-----END PGP SIGNATURE-----`

var validMessageBody = `
-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256

{"type": "test"}
-----BEGIN PGP SIGNATURE-----

iQFVBAEBCAA/FiEE/miAUJRHb5lzwQGz4j1MCbB13EwFAlqVcqIhHGRldi5kaWdp
dGFscHVibGlzaGluZ0BvbnMuZ292LnVrAAoJEOI9TAmwddxMO8YIALc1wfrkMbrw
zsTIhskId/5CPPKTLYfLprZESFU074p6pd3ySRcKNkc+rglwTk9ZJBITtFHE6ORP
BlEZXHoCZjzfxidySCedNZnMGtuQfucf3q1UiF2Wl9vQ0huVdcgJuK1oDSkNACd3
xPhiEvRPaJeKjXOtT/v9Z0FkqsRSLLk5/j7rMUhQ5JtocHejYMyYid8OyC+zo/Pu
jKdQA/HV//LSSfLicUCwdC+NkY/EWKcChgr0cQBYeVYJ0Aw5n/NYn/4qzFzCnh9s
DKnWELoSVCH7/5OEbiNvWm5Id8NyXmtzhw+076t3h5Za3OKmh3aZjSl4IpweDV7m
cKHPGDHOlrA=
=YeHf
-----END PGP SIGNATURE-----`

var invalidMessageBody = `
-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA256

{"type": "test"}
-----BEGIN PGP SIGNATURE-----

iQFPBAEBCAA5FiEErw3YMMpYucPRmd7DgcrrB3p/OiMFAlqWWKUbHGxsb3lkLmdy
aWZmaXRoc0BvbnMuZ292LnVrAAoJEIHK6wd6fzojhk4H/3749Jr0st/nW9nJS6WD
P3ZfF1t1do0kI9I1QPlyIF19YboNSsaCEeh6DgvR6g0MW25a4DUcVzSfQsdp//p4
XV89u1KqKNvNrgciS55h/a/s+gR86Hsm7QGop31GBlKmEJSUP6N8frMRKFd0KSP6
pqSJMT+yx8rrOUldiWdjqCO7RvUvUFbrnaj4DqC5L7e2KcyG6ssCvsK4bJS9j/aI
2WHK7l0VksZEEr1u1rD21VgT+Uk5LQfNViW5ca0n6DAgM9g+XYEVecU2+aaGfzok
bevQkIOyqy56gbWrpMqM294xuKFC/hhpQaL1IBmsexLKsfwZ04dKqBKR/viIOX1R
YH0=
=v09o
-----END PGP SIGNATURE-----`

var (
	emptyMessage = &sqs.Message{
		MessageId:     aws.String("100"),
		ReceiptHandle: aws.String(""),
		Body:          aws.String(emptyMessageBody),
	}

	validMessage = &sqs.Message{
		MessageId:     aws.String("200"),
		ReceiptHandle: aws.String(""),
		Body:          aws.String(validMessageBody),
	}

	invalidMessage = &sqs.Message{
		MessageId:     aws.String("400"),
		ReceiptHandle: aws.String(""),
		Body:          aws.String(invalidMessageBody),
	}

	unsignedMessage = &sqs.Message{
		MessageId:     aws.String("300"),
		ReceiptHandle: aws.String(""),
		Body:          aws.String(`{"type": "test"}`),
	}
)

var defaultErrHandler = ErrHandler

func TestNew(t *testing.T) {
	os.Clearenv()
	os.Setenv("AWS_CREDENTIAL_FILE", "/i/hope/this/path/does/not/exist")
	defer os.Unsetenv("AWS_CREDENTIAL_FILE")

	fixtures := []struct {
		config   *Config
		errMsg   string
		isPrefix bool
	}{
		{
			&Config{
				ConsumerQueue:    "",
				ConsumerQueueURL: "foo",
				ProducerQueue:    "bar",
				Region:           "baz",
				VerificationKey:  publicKey,
			},
			"missing consumer queue name",
			false,
		},
		{
			&Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "",
				ProducerQueue:    "bar",
				Region:           "baz",
				VerificationKey:  publicKey,
			},
			"missing consumer queue url",
			false,
		},
		{
			&Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "bar",
				ProducerQueue:    "",
				Region:           "baz",
				VerificationKey:  publicKey,
			},
			"missing producer queue name",
			false,
		},
		{
			&Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "bar",
				ProducerQueue:    "baz",
				Region:           "",
				VerificationKey:  publicKey,
			},
			"missing queue region",
			false,
		},
		{
			&Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "bar",
				ProducerQueue:    "baz",
				Region:           "qux",
				VerificationKey:  publicKey,
			},
			"No valid AWS authentication found",
			true,
		},
		{
			&Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "bar",
				ProducerQueue:    "baz",
				Region:           "qux",
				VerificationKey:  "",
			},
			"openpgp: invalid argument: no armored data found",
			false,
		},
	}

	for _, fixture := range fixtures {
		Convey("an error is returned with invalid configuration", t, func() {
			e, err := New(fixture.config, nil)
			So(e, ShouldBeNil)
			So(err, ShouldNotBeNil)
			if fixture.isPrefix {
				So(err.Error(), ShouldStartWith, fixture.errMsg)
			} else {
				So(err.Error(), ShouldEqual, fixture.errMsg)
			}
		})
	}

	withEnv(func() {
		Convey("an engine is returned with valid configuration", t, func() {
			config := &Config{
				ConsumerQueue:    "foo",
				ConsumerQueueURL: "bar",
				ProducerQueue:    "baz",
				Region:           "qux",
				VerificationKey:  publicKey,
			}

			e, err := New(config, nil)
			So(err, ShouldBeNil)
			So(e, ShouldNotBeNil)
		})
	})
}

func TestStart(t *testing.T) {
	withEnv(func() {
		Convey("start functions as expected", t, func(c C) {
			ctx, cancel := context.WithCancel(context.Background())

			doErrTest := func(handlers map[string]HandlerFunc, errorable bool, consumedMsg *sqs.Message, producedMsgID, producedMsgBody, engineErr string) {
				withMocks(errorable, consumedMsg, func(producer *mockProducer) {
					e, err := New(&Config{"foo", "bar", "baz", "qux", publicKey}, handlers)
					So(e, ShouldNotBeNil)
					So(err, ShouldBeNil)

					ErrHandler = func(messageID string, err error) {
						cancel()
						c.So(messageID, ShouldEqual, producedMsgID)
						c.So(err.Error(), ShouldEqual, engineErr)
					}

					e.Start(ctx)
					c.So(producer.message, ShouldEqual, producedMsgBody)
				})
			}

			Convey("queue errors are propogated as expected", func() {
				expectedError := "consumer error"
				expectedMsgID := ""
				expectedMsgBody := ""
				doErrTest(nil, true, invalidMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("unmarshaling errors are propogated as expected", func() {
				expectedError := "unexpected end of JSON input"
				expectedMsgID := "100"
				expectedMsgBody := `{"Error":{"Data":{"Offset":1},"Message":"unexpected end of JSON input"},"ID":"100","Success":false}`
				doErrTest(nil, false, emptyMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("missing handler errors are propogated as expected", func() {
				expectedError := "missing handler for message"
				expectedMsgID := "200"
				expectedMsgBody := `{"Error":{"Data":{"MessageType":"test"},"Message":"missing handler for message"},"ID":"200","Success":false}`
				doErrTest(nil, false, validMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("handler errors are propogated as expected", func() {
				handlers := map[string]HandlerFunc{
					"test": func(ctx context.Context, msg *Message) error { return &handlerError{"foo", "bar"} },
				}
				expectedError := "handler error"
				expectedMsgID := "200"
				expectedMsgBody := `{"Error":{"Data":{"Field1":"foo","Field2":"bar"},"Message":"handler error"},"ID":"200","Success":false}`
				doErrTest(handlers, false, validMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("unsigned message errors are propogated as expected", func() {
				expectedError := "invalid clearsign block for message"
				expectedMsgID := "300"
				expectedMsgBody := `{"Error":{"Data":{"MessageID":"300"},"Message":"invalid clearsign block for message"},"ID":"300","Success":false}`
				doErrTest(nil, false, unsignedMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("invalid signature errors are propogated as expected", func() {
				expectedError := "openpgp: signature made by unknown entity"
				expectedMsgID := "400"
				expectedMsgBody := `{"Error":{"Data":0,"Message":"openpgp: signature made by unknown entity"},"ID":"400","Success":false}`
				doErrTest(nil, false, invalidMessage, expectedMsgID, expectedMsgBody, expectedError)
			})

			Convey("successful message handles are propogated as expected", func() {
				withMocks(false, validMessage, func(producer *mockProducer) {
					e, err := New(&Config{"foo", "bar", "baz", "qux", publicKey}, nil)
					So(e, ShouldNotBeNil)
					So(err, ShouldBeNil)

					hfunction := func(ctx context.Context, msg *Message) error { return nil }
					e.handlers = map[string]HandlerFunc{"test": hfunction}
					ErrHandler = defaultErrHandler

					go time.AfterFunc(time.Second*1, cancel)
					e.Start(ctx)
					So(producer.message, ShouldEqual, `{"ID":"200","Success":true}`)
				})
			})
		})
	})
}

func withEnv(f func()) {
	defer os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "FOO")
	os.Setenv("AWS_DEFAULT_REGION", "BAR")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "BAZ")
	f()
}

type mockConsumer struct {
	exhausted bool
	sqsiface.ClientAPI
	errorable bool
	message   *sqs.Message
	mu        sync.Mutex
}

type mockProducer struct {
	message string
}

func (m *mockConsumer) ReceiveMessage(in *sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error) {
	m.mu.Lock()

	defer func() {
		m.exhausted = true
		m.mu.Unlock()
	}()

	if m.exhausted {
		return &sqs.ReceiveMessageOutput{Messages: nil}, nil
	}
	if m.errorable {
		return nil, errors.New("consumer error")
	}
	return &sqs.ReceiveMessageOutput{Messages: []sqs.Message{*m.message}}, nil
}

func (m *mockConsumer) DeleteMessage(in *sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error) {
	return nil, nil
}

func (m *mockProducer) SendMessage(body string) error {
	m.message = body
	return nil
}

func withMocks(errorable bool, msg *sqs.Message, f func(*mockProducer)) {
	defaultClient := ssqs.DefaultClient
	defaultSendMsg := sendMessage

	defer func() {
		sendMessage = defaultSendMsg
		ssqs.DefaultClient = defaultClient
	}()

	ssqs.DefaultClient = func(c aws.Config) sqsiface.ClientAPI {
		return &mockConsumer{errorable: errorable, message: msg}
	}

	mockProducer := &mockProducer{}
	sendMessage = mockProducer.SendMessage
	f(mockProducer)
}
