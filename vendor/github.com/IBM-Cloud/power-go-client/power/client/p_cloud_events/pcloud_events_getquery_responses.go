// Code generated by go-swagger; DO NOT EDIT.

package p_cloud_events

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	models "github.com/IBM-Cloud/power-go-client/power/models"
)

// PcloudEventsGetqueryReader is a Reader for the PcloudEventsGetquery structure.
type PcloudEventsGetqueryReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *PcloudEventsGetqueryReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewPcloudEventsGetqueryOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 400:
		result := NewPcloudEventsGetqueryBadRequest()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	case 500:
		result := NewPcloudEventsGetqueryInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewPcloudEventsGetqueryOK creates a PcloudEventsGetqueryOK with default headers values
func NewPcloudEventsGetqueryOK() *PcloudEventsGetqueryOK {
	return &PcloudEventsGetqueryOK{}
}

/*PcloudEventsGetqueryOK handles this case with default header values.

OK
*/
type PcloudEventsGetqueryOK struct {
	Payload *models.Events
}

func (o *PcloudEventsGetqueryOK) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/events][%d] pcloudEventsGetqueryOK  %+v", 200, o.Payload)
}

func (o *PcloudEventsGetqueryOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Events)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudEventsGetqueryBadRequest creates a PcloudEventsGetqueryBadRequest with default headers values
func NewPcloudEventsGetqueryBadRequest() *PcloudEventsGetqueryBadRequest {
	return &PcloudEventsGetqueryBadRequest{}
}

/*PcloudEventsGetqueryBadRequest handles this case with default header values.

Bad Request
*/
type PcloudEventsGetqueryBadRequest struct {
	Payload *models.Error
}

func (o *PcloudEventsGetqueryBadRequest) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/events][%d] pcloudEventsGetqueryBadRequest  %+v", 400, o.Payload)
}

func (o *PcloudEventsGetqueryBadRequest) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewPcloudEventsGetqueryInternalServerError creates a PcloudEventsGetqueryInternalServerError with default headers values
func NewPcloudEventsGetqueryInternalServerError() *PcloudEventsGetqueryInternalServerError {
	return &PcloudEventsGetqueryInternalServerError{}
}

/*PcloudEventsGetqueryInternalServerError handles this case with default header values.

Internal Server Error
*/
type PcloudEventsGetqueryInternalServerError struct {
	Payload *models.Error
}

func (o *PcloudEventsGetqueryInternalServerError) Error() string {
	return fmt.Sprintf("[GET /pcloud/v1/cloud-instances/{cloud_instance_id}/events][%d] pcloudEventsGetqueryInternalServerError  %+v", 500, o.Payload)
}

func (o *PcloudEventsGetqueryInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
