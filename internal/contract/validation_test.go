package contract_test

import (
	"testing"

	"github.com/Scale-Flow/trello-cli/internal/contract"
)

func TestRequireFlagPresent(t *testing.T) {
	err := contract.RequireFlag("board", "abc123")
	if err != nil {
		t.Errorf("RequireFlag() with value should return nil, got %v", err)
	}
}

func TestRequireFlagMissing(t *testing.T) {
	err := contract.RequireFlag("board", "")
	if err == nil {
		t.Fatal("RequireFlag() with empty value should return error")
	}

	ce, ok := err.(*contract.ContractError)
	if !ok {
		t.Fatal("RequireFlag() should return *ContractError")
	}
	if ce.Code != contract.ValidationError {
		t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
	}
}

func TestRequireExactlyOneWithOne(t *testing.T) {
	err := contract.RequireExactlyOne(map[string]string{
		"board": "abc",
		"list":  "",
	})
	if err != nil {
		t.Errorf("RequireExactlyOne() with one value should return nil, got %v", err)
	}
}

func TestRequireExactlyOneWithNone(t *testing.T) {
	err := contract.RequireExactlyOne(map[string]string{
		"board": "",
		"list":  "",
	})
	if err == nil {
		t.Fatal("RequireExactlyOne() with no values should return error")
	}
	ce := err.(*contract.ContractError)
	if ce.Code != contract.ValidationError {
		t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
	}
}

func TestRequireExactlyOneWithTwo(t *testing.T) {
	err := contract.RequireExactlyOne(map[string]string{
		"board": "abc",
		"list":  "def",
	})
	if err == nil {
		t.Fatal("RequireExactlyOne() with two values should return error")
	}
	ce := err.(*contract.ContractError)
	if ce.Code != contract.ValidationError {
		t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
	}
}

func TestRequireAtLeastOneWithOne(t *testing.T) {
	err := contract.RequireAtLeastOne(map[string]string{
		"name": "new name",
		"pos":  "",
	})
	if err != nil {
		t.Errorf("RequireAtLeastOne() with one value should return nil, got %v", err)
	}
}

func TestRequireAtLeastOneWithNone(t *testing.T) {
	err := contract.RequireAtLeastOne(map[string]string{
		"name": "",
		"pos":  "",
	})
	if err == nil {
		t.Fatal("RequireAtLeastOne() with no values should return error")
	}
	ce := err.(*contract.ContractError)
	if ce.Code != contract.ValidationError {
		t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
	}
}

func TestRequireAtLeastOneWithAll(t *testing.T) {
	err := contract.RequireAtLeastOne(map[string]string{
		"name": "new name",
		"pos":  "top",
	})
	if err != nil {
		t.Errorf("RequireAtLeastOne() with all values should return nil, got %v", err)
	}
}

func TestValidateISO8601Valid(t *testing.T) {
	valid := []string{
		"2026-03-20T00:00:00.000Z",
		"2026-03-20",
		"2026-12-31T23:59:59Z",
	}
	for _, v := range valid {
		if err := contract.ValidateISO8601(v); err != nil {
			t.Errorf("ValidateISO8601(%q) should be valid, got %v", v, err)
		}
	}
}

func TestValidateISO8601Invalid(t *testing.T) {
	invalid := []string{
		"not-a-date",
		"03/20/2026",
		"",
	}
	for _, v := range invalid {
		err := contract.ValidateISO8601(v)
		if err == nil {
			t.Errorf("ValidateISO8601(%q) should be invalid", v)
			continue
		}
		ce := err.(*contract.ContractError)
		if ce.Code != contract.ValidationError {
			t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
		}
	}
}

func TestValidateISO8601Empty(t *testing.T) {
	// Empty string means the flag was not provided — skip validation
	err := contract.ValidateISO8601Optional("")
	if err != nil {
		t.Errorf("ValidateISO8601Optional(\"\") should return nil, got %v", err)
	}
}

func TestValidateURLValid(t *testing.T) {
	err := contract.ValidateURL("https://example.com/file.pdf")
	if err != nil {
		t.Errorf("ValidateURL() with valid URL should return nil, got %v", err)
	}
}

func TestValidateURLInvalid(t *testing.T) {
	invalid := []string{
		"not-a-url",
		"",
		"ftp://nope",
	}
	for _, v := range invalid {
		err := contract.ValidateURL(v)
		if err == nil {
			t.Errorf("ValidateURL(%q) should be invalid", v)
			continue
		}
		ce := err.(*contract.ContractError)
		if ce.Code != contract.ValidationError {
			t.Errorf("Code = %q, want %q", ce.Code, contract.ValidationError)
		}
	}
}

func TestValidateFilePathValid(t *testing.T) {
	// Use a file that definitely exists
	err := contract.ValidateFilePath("/dev/null")
	if err != nil {
		t.Errorf("ValidateFilePath() with existing file should return nil, got %v", err)
	}
}

func TestValidateFilePathMissing(t *testing.T) {
	err := contract.ValidateFilePath("/nonexistent/file/path.txt")
	if err == nil {
		t.Fatal("ValidateFilePath() with missing file should return error")
	}
	ce := err.(*contract.ContractError)
	if ce.Code != contract.FileNotFound {
		t.Errorf("Code = %q, want %q", ce.Code, contract.FileNotFound)
	}
}

func TestValidateFilePathEmpty(t *testing.T) {
	err := contract.ValidateFilePath("")
	if err == nil {
		t.Fatal("ValidateFilePath() with empty path should return error")
	}
	ce := err.(*contract.ContractError)
	if ce.Code != contract.FileNotFound {
		t.Errorf("Code = %q, want %q", ce.Code, contract.FileNotFound)
	}
}
