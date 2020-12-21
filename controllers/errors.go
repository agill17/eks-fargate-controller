package controllers

type ErrEksClusterNotFound struct {
	Message string
}

func (e ErrEksClusterNotFound) Error() string {
	return e.Message
}

type ErrEksClusterNotActive struct {
	Message string
}

func (e ErrEksClusterNotActive) Error() string {
	return e.Message
}

type ErrInvalidSubnet struct {
	Message string
}

func (e ErrInvalidSubnet) Error() string {
	return e.Message
}
