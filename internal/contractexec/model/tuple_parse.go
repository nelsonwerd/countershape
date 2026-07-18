package model

import (
	"encoding/base64"

	"github.com/nelsonwerd/countershape/internal/canon"
	emitmodel "github.com/nelsonwerd/countershape/internal/emit/node/model"
	"github.com/nelsonwerd/countershape/internal/portablevalue"
)

func parseExactTupleValue(value canon.Value) (emitmodel.ExactTuple, error) {
	if err := requireRoster(value, "fields"); err != nil {
		return emitmodel.ExactTuple{}, refuse(CodeInvalidWitness, "observed tuple root roster differs", err)
	}
	entries, err := arrayMember(value, "fields")
	if err != nil {
		return emitmodel.ExactTuple{}, err
	}
	fields := make([]emitmodel.ExactField, len(entries))
	for index, entry := range entries {
		if err := requireRoster(entry, "field_id", "value"); err != nil {
			return emitmodel.ExactTuple{}, refuse(CodeInvalidWitness, "observed tuple field roster differs", err)
		}
		fieldID, err := textMember(entry, "field_id")
		if err != nil {
			return emitmodel.ExactTuple{}, err
		}
		valueNode, err := member(entry, "value")
		if err != nil {
			return emitmodel.ExactTuple{}, err
		}
		portable, err := parsePortableValue(valueNode)
		if err != nil {
			return emitmodel.ExactTuple{}, err
		}
		exact, err := emitmodel.NewExactValue(portable)
		if err != nil {
			return emitmodel.ExactTuple{}, refuse(CodeInvalidWitness, "observed tuple value is invalid", err)
		}
		fields[index], err = emitmodel.NewExactField(fieldID, exact)
		if err != nil {
			return emitmodel.ExactTuple{}, refuse(CodeInvalidWitness, "observed tuple field is invalid", err)
		}
	}
	tuple, err := emitmodel.NewExactTuple(fields)
	if err != nil {
		return emitmodel.ExactTuple{}, refuse(CodeInvalidWitness, "observed tuple is invalid", err)
	}
	return tuple, nil
}

func parsePortableValue(value canon.Value) (portablevalue.Value, error) {
	tag, err := textMember(value, "tag")
	if err != nil {
		return portablevalue.Value{}, err
	}
	switch tag {
	case string(portablevalue.TagMissing):
		if err := requireRoster(value, "tag"); err != nil {
			return portablevalue.Value{}, err
		}
		return portablevalue.Missing(), nil
	case string(portablevalue.TagNull):
		if err := requireRoster(value, "tag"); err != nil {
			return portablevalue.Value{}, err
		}
		return portablevalue.Null(), nil
	case string(portablevalue.TagBoolean):
		if err := requireRoster(value, "tag", "value"); err != nil {
			return portablevalue.Value{}, err
		}
		boolean, err := boolMember(value, "value")
		if err != nil {
			return portablevalue.Value{}, err
		}
		return portablevalue.Boolean(boolean), nil
	case string(portablevalue.TagString):
		if err := requireRoster(value, "tag", "value"); err != nil {
			return portablevalue.Value{}, err
		}
		text, err := textMember(value, "value")
		if err != nil {
			return portablevalue.Value{}, err
		}
		return portablevalue.String(text)
	case string(portablevalue.TagInteger):
		if err := requireRoster(value, "tag", "canonical"); err != nil {
			return portablevalue.Value{}, err
		}
		integer, err := textMember(value, "canonical")
		if err != nil {
			return portablevalue.Value{}, err
		}
		return portablevalue.Integer(integer)
	case string(portablevalue.TagBytes):
		if err := requireRoster(value, "tag", "base64"); err != nil {
			return portablevalue.Value{}, err
		}
		encoded, err := textMember(value, "base64")
		if err != nil {
			return portablevalue.Value{}, err
		}
		opaque, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil || base64.StdEncoding.EncodeToString(opaque) != encoded {
			return portablevalue.Value{}, refuse(CodeInvalidWitness, "observed bytes use noncanonical base64", err)
		}
		return portablevalue.Bytes(opaque)
	case string(portablevalue.TagOrderedStringList):
		if err := requireRoster(value, "tag", "values"); err != nil {
			return portablevalue.Value{}, err
		}
		entries, err := arrayMember(value, "values")
		if err != nil {
			return portablevalue.Value{}, err
		}
		values := make([]string, len(entries))
		for index, entry := range entries {
			text, ok := entry.Text()
			if !ok {
				return portablevalue.Value{}, refuse(CodeInvalidWitness, "ordered string list contains a non-string", nil)
			}
			values[index] = text
		}
		return portablevalue.OrderedStringList(values)
	case string(portablevalue.TagCanonicalJSON):
		if err := requireRoster(value, "tag", "canonical_base64"); err != nil {
			return portablevalue.Value{}, err
		}
		encoded, err := textMember(value, "canonical_base64")
		if err != nil {
			return portablevalue.Value{}, err
		}
		exact, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil || base64.StdEncoding.EncodeToString(exact) != encoded {
			return portablevalue.Value{}, refuse(CodeInvalidWitness, "canonical JSON uses noncanonical base64", err)
		}
		return portablevalue.CanonicalJSON(exact)
	default:
		return portablevalue.Value{}, refuse(CodeInvalidWitness, "observed tuple value tag is unknown", nil)
	}
}
