package marketplace

import "testing"

func TestProvider(t *testing.T) {
    if err := Provider().InternalValidate(); err != nil {
        t.Error(err)
    }
}

