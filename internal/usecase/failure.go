package usecase

import commonentity "github.com/xaligo/terraform-provider/internal/entity/common"

func fail(summary string, err error) error {
	return &commonentity.Failure{Summary: summary, Err: err}
}
