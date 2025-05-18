package di

import (
	"fmt"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pixie-sh/errors-go"
)

const (
	timeEncodingFormat = time.RFC3339Nano
)

func isPointer(i interface{}) bool {
	if i == nil {
		return false
	}

	return reflect.TypeOf(i).Kind() == reflect.Ptr
}

func DecodeStruct(from any, to any) error {
	if !isPointer(to) {
		return errors.New("destination must be pointer", StructMapTypeMismatchErrorCode)
	}

	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				stringToTimeHook,
				timeToStringHook,
			),
			TagName: "json",
			Result:  to,
		})
	if err != nil {
		return errors.Wrap(err, "failed to create decoder", StructMapTypeMismatchErrorCode)
	}

	err = decoder.Decode(from)
	if err != nil {
		return errors.Wrap(err, "failed to decode", StructMapTypeMismatchErrorCode)
	}

	return nil
}

func Decode[T any](from any) (T, error) {
	var to T
	return to, DecodeStruct(from, &to)
}

func stringToTimeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f == reflect.TypeOf("") && t == reflect.TypeOf(time.Time{}) {
		parsedTime, err := time.Parse(timeEncodingFormat, data.(string))
		if err != nil {
			return nil, err
		}
		return parsedTime, nil
	}

	if f == reflect.TypeOf(map[string]interface{}{}) && t == reflect.TypeOf(time.Time{}) {
		dataCasted, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("data is not a map")
		}

		timeStr, ok := dataCasted["RFC3339"].(string)
		if !ok {
			return nil, fmt.Errorf("RFC3339 key not found or not a string")
		}

		parsedTime, err := time.Parse(timeEncodingFormat, timeStr)
		if err != nil {
			return nil, err
		}
		return parsedTime, nil
	}

	return data, nil
}

func timeToStringHook(f reflect.Type, _ reflect.Type, data interface{}) (interface{}, error) {
	if f == reflect.TypeOf(&time.Time{}) {
		return serializeTimeToMap(data.(*time.Time)), nil
	}

	return data, nil
}

func serializeTimeToMap(t *time.Time) map[string]string {
	return map[string]string{
		"RFC3339": t.UTC().Format(timeEncodingFormat),
	}
}
