// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsutil

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	secrets "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"
)

// InterpretError attempts to interpret AWS SDK errors and either improve
// the error reporting to the caller and/or map to already defined error
// types as fs.ErrNotExist.
//
// secretmanager.ResourceNotFoundException is mapped to fs.ErrNotExist, and
// secretmanager.InvalidRequestException with "currently marked deleted" in the
// message is also mapped to fs.ErrNotExist, as the secret is not accessible.
//
// The error message "security token included in the request is invalid" can
// be caused by multiple issues, such as an incorrect Secret Access Key,
// an expired Session Token (very common with IAM roles/temporary credentials),
// or an incorrect Access Key ID. This is interpreted and the returned
// error is wrapped with a hint to check AWS credentials/configuration.
func InterpretError(err error) error {
	if err == nil {
		return nil
	}
	if serr, ok := errors.AsType[*secrets.ResourceNotFoundException](err); ok {
		return fmt.Errorf("%v: %w", serr.Error(), fs.ErrNotExist)
	}
	if serr, ok := errors.AsType[*secrets.InvalidRequestException](err); ok {
		if strings.Contains(serr.ErrorMessage(), "currently marked deleted") {
			return fmt.Errorf("%v: %w", serr.Error(), fs.ErrNotExist)
		}
		return err
	}
	if opre, ok := errors.AsType[*smithy.OperationError](err); ok {
		if strings.Contains(opre.Error(), "security token included in the request is invalid") {
			return fmt.Errorf("%w (check AWS credentials/configuration)", err)
		}
	}
	return err
}
