// Code generated by goa v3.10.2, DO NOT EDIT.
//
// catalog HTTP client encoders and decoders
//
// Command:
// $ goa gen github.com/tektoncd/hub/api/v1/design

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"

	catalog "github.com/tektoncd/hub/api/v1/gen/catalog"
	goahttp "goa.design/goa/v3/http"
)

// BuildListRequest instantiates a HTTP request object with method and path set
// to call the "catalog" service "List" endpoint
func (c *Client) BuildListRequest(ctx context.Context, v interface{}) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ListCatalogPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("catalog", "List", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeListResponse returns a decoder for responses returned by the catalog
// List endpoint. restoreBody controls whether the response body should be
// restored after having been read.
// DecodeListResponse may return the following errors:
//	- "internal-error" (type *goa.ServiceError): http.StatusInternalServerError
//	- error: internal error
func DecodeListResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (interface{}, error) {
	return func(resp *http.Response) (interface{}, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body ListResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("catalog", "List", err)
			}
			err = ValidateListResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("catalog", "List", err)
			}
			res := NewListResultOK(&body)
			return res, nil
		case http.StatusInternalServerError:
			var (
				body ListInternalErrorResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("catalog", "List", err)
			}
			err = ValidateListInternalErrorResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("catalog", "List", err)
			}
			return nil, NewListInternalError(&body)
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("catalog", "List", resp.StatusCode, string(body))
		}
	}
}

// unmarshalCatalogResponseBodyToCatalogCatalog builds a value of type
// *catalog.Catalog from a value of type *CatalogResponseBody.
func unmarshalCatalogResponseBodyToCatalogCatalog(v *CatalogResponseBody) *catalog.Catalog {
	if v == nil {
		return nil
	}
	res := &catalog.Catalog{
		ID:       *v.ID,
		Name:     *v.Name,
		Type:     *v.Type,
		URL:      *v.URL,
		Provider: *v.Provider,
	}

	return res
}
