package secret

import (
	"io"
	"os"
	"strings"
	"testing"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	. "github.com/smartystreets/goconvey/convey"
)

var testMessage = `-----BEGIN PGP MESSAGE-----

hQEMA48Y0Zt/+vrbAQf+Ka5lMaIhBL63DcQvIRf6ozwxXag5lvhcjnDWkJqjjWSr
BXgY8TQUoWtsDcC2eN80Fluv3oLsRxTwzxvImKzm4AQ26zAgCUlD57RJ4pk/H7PT
CDgISl3anYyu1wghljcbTr69JQGmt2UlsY6TXrsi5mcG/jIrwCsoQHKSvqzksnSJ
7NBC/N5Ld+d90Kv5bv53h6uaIRP+q7DMDSO0AGneSlPKxI2Les6R8+iCa8gAa0PT
CjXyRcHWKW75S3i5G1nIE9QUf0JzwOkdFM9ycKCz3xxsSLzC6xmavM3DZBoe0SVr
UfW7K5+z/Hlp2EDjmzVrQEkOKKi9Vc+gFrKnONgS+tJcARvxoWQlVu7T8iawXE1f
5JX2TzkUgKKiPRUbGBuNKEjCrp9R/yrxm2spFHfw5Sv6fUOfyDeVW5KIXMU38yNS
qqFh4AwsD5fkANp8Rmwwx0woYdsVY0WOdAoF9Xk=
=Dvb6
-----END PGP MESSAGE-----`

var testPrivateKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

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
Z/E5hX0lDHzAklyBHfVeUdarqA==
=1vcd
-----END PGP PRIVATE KEY BLOCK-----`

func stringKeyReader(str string) func() (io.Reader, error) {
	return func() (io.Reader, error) { return strings.NewReader(str), nil }
}

func TestNew(t *testing.T) {
	if testing.Short() {
		t.Skip("short test run - skipping")
	}

	withMocks(func() {
		Convey("an error is returned with misconfiguration", t, func() {
			s, err := New(&Config{"", "foo"})
			So(s, ShouldBeNil)
			So(err, ShouldNotBeNil)
			So(err.Error(), ShouldEqual, "No valid AWS authentication found")
		})
	})

	withEnv(func() {
		withMocks(func() {
			Convey("a handler is returned with good configuration", t, func() {
				s, err := New(&Config{"", "bar"})
				So(err, ShouldBeNil)
				So(s, ShouldNotBeNil)
			})
		})
	})
}

func TestEntity(t *testing.T) {
	withEnv(func() {
		withMocks(func() {
			Convey("successfully creates openpgp entity", t, func() {
				s, err := New(&Config{"", "foo"})
				So(err, ShouldBeNil)
				So(s, ShouldNotBeNil)

				e, err := entityList(strings.NewReader(testPrivateKey))
				So(err, ShouldBeNil)
				So(e, ShouldNotBeNil)
				So(len(e.DecryptionKeys()), ShouldEqual, 1)
			})
		})
	})
}

func TestDearmor(t *testing.T) {
	withEnv(func() {
		withMocks(func() {
			Convey("successfully strips armor", t, func() {
				s, err := New(&Config{"", "eu-west-1"})
				So(err, ShouldBeNil)
				So(s, ShouldNotBeNil)

				for _, v := range []string{testMessage, testPrivateKey} {
					m, err := dearmorMessage(strings.NewReader(v))
					So(err, ShouldBeNil)
					So(m, ShouldNotBeNil)
				}
			})
		})
	})
}

func TestDecrypt(t *testing.T) {
	withEnv(func() {
		withMocks(func() {
			Convey("successfully decrypts message", t, func() {
				s, err := New(&Config{"", "eu-west-1"})
				So(err, ShouldBeNil)
				So(s, ShouldNotBeNil)

				m, err := s.decryptMessage([]byte(testMessage))
				So(err, ShouldBeNil)
				So(m, ShouldNotBeNil)
				So(string(m), ShouldStartWith, `{ "message": "hello world" }`)
			})
		})
	})
}

func TestWrite(t *testing.T) {
	withEnv(func() {
		withMocks(func() {
			Convey("wite behaives correctly", t, func() {
				s, err := New(&Config{"", "eu-west-1"})
				So(err, ShouldBeNil)
				So(s, ShouldNotBeNil)

				m, err := s.decryptMessage([]byte(testMessage))
				So(err, ShouldBeNil)
				So(m, ShouldNotBeNil)

				httpmock.DeactivateAndReset()
				httpmock.ActivateNonDefault(s.vaultHTTPClient)

				Convey("writes secret correctly", func() {
					httpmock.RegisterResponder("PUT", "http://localhost:8200/v1/secret/test", httpmock.NewStringResponder(200, "{}"))
					err := s.write("test", m)
					So(err, ShouldBeNil)
				})

				Convey("handles error correctly", func() {
					httpmock.RegisterResponder("PUT", "http://localhost:8200/v1/secret/test", httpmock.NewStringResponder(401, "{}"))
					err := s.write("test", m)
					So(err, ShouldNotBeNil)
					So(err.Error(), ShouldStartWith, "Error making API request")
				})
			})
		})
	})
}

func withEnv(f func()) {
	defer os.Clearenv()

	os.Clearenv()
	os.Setenv("AWS_ACCESS_KEY_ID", "FOO")
	os.Setenv("AWS_DEFAULT_REGION", "BAR")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "BAZ")
	os.Setenv("VAULT_ADDR", "http://localhost:8200")

	f()
}

func withMocks(f func()) {
	origJSONFrom := keyReader

	defer func() {
		keyReader = origJSONFrom
	}()

	keyReader = stringKeyReader(testPrivateKey)
	f()
}
