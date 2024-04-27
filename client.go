package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type Client struct {
	c          *Config
	httpClient *http.Client
}

func NewClient(config *Config, opts ...ClientOpts) *Client {
	m := &Client{
		c:          config,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type ClientOpts func(*Client)

func WithCustomHTTPClient(c *http.Client) ClientOpts {
	return func(m *Client) {
		m.httpClient = c
	}
}

func (c *Client) Send(req *http.Request, v interface{}) error {
	req.Header = http.Header{
		"Content-type":  {"application/json"},
		"Authorization": {"Basic " + c.c.BasicAuth},
	}

	if c.c.Debug {
		log.Printf(
			"[RozetkaPay] Debug --- type: %s, method: %s, url: %s\n",
			"request",
			req.Method,
			req.URL.String(),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var errResp *ErrorResponse
		if len(body) == 0 {
			return ErrResponseIsEmpty
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			return err
		}

		log.Printf(
			"[RozetkaPay] Error --- type: %s, code: %s, message: %s, payment_id: %s, type: %s\n",
			errResp.Type,
			errResp.Code,
			errResp.Message,
			errResp.PaymentID,
			errResp.Type,
		)

		return errResp.ErrorCode()
	}

	if v == nil {
		return nil
	}

	if c.c.Debug {
		log.Printf(
			"[RozetkaPay] Debug --- type: %s, method: %s, url: %s, code: %d, bytes: %d\n",
			"response",
			req.Method,
			req.URL.String(),
			resp.StatusCode,
			len(body),
		)
	}

	return json.Unmarshal(body, v)
}

func (c *Client) NewRequest(method, url string, payload interface{}, query map[string]string) (
	*http.Request, error,
) {
	var buf io.Reader
	if payload != nil {
		b, err := json.Marshal(&payload)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	return req, nil
}

// Creates payment and performs desired operation.
func (c *Client) CreatePayment(schema *CreatePaymentSchema) (*PaymentResponse, error) {
	req, err := c.NewRequest(http.MethodPost, c.c.API+"payments/v1/new", schema, nil)
	if err != nil {
		return nil, err
	}
	resp := &PaymentResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Confirm two-step payment.
func (c *Client) ConfirmPayment(schema *ConfirmPaymentSchema) (*PaymentResponse, error) {
	req, err := c.NewRequest(http.MethodPost, c.c.API+"payments/v1/confirm", schema, nil)
	if err != nil {
		return nil, err
	}
	resp := &PaymentResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Cancel two-step payment.
func (c *Client) CancelPayment(schema *CancelPaymentSchema) (*PaymentResponse, error) {
	req, err := c.NewRequest(http.MethodPost, c.c.API+"payments/v1/cancel", schema, nil)
	if err != nil {
		return nil, err
	}
	resp := &PaymentResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Refund one-step payment after withdrawal, or two-step payment after confirmation.
func (c *Client) RefundPayment(schema *RefundPaymentSchema) (*PaymentResponse, error) {
	req, err := c.NewRequest(http.MethodPost, c.c.API+"payments/v1/refund", schema, nil)
	if err != nil {
		return nil, err
	}
	resp := &PaymentResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Get payment info by id.
func (c *Client) GetPaymentInfo(externalID string) (*PaymentInfoResponse, error) {
	req, err := c.NewRequest(
		http.MethodGet, c.c.API+"payments/v1/info",
		nil, map[string]string{"external_id": externalID},
	)
	if err != nil {
		return nil, err
	}
	resp := &PaymentInfoResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Parsing callback from the body.
func (c *Client) GetPaymentCallbackFromBytes(body []byte) (*PaymentResponse, error) {
	var callback *PaymentResponse
	if err := json.Unmarshal(body, &callback); err != nil {
		return nil, err
	}
	return callback, nil
}

// Prepares the data about the specified payment of transaction and sends it into callback_url which was provided on the payment step.
// If the operation field is not provided the callback will be sent for the last operation.
func (c *Client) ResendPaymentCallback(schema *PaymentCallbackResendSchema) (resended bool, err error) {
	req, err := c.NewRequest(http.MethodPost, c.c.API+"payments/v1/callback/resend", schema, nil)
	if err != nil {
		return false, err
	}
	if err := c.Send(req, nil); err != nil {
		return false, err
	}
	return true, nil
}

// Adds new payment method to wallet.
func (c *Client) AddWalletCustomerPayment(customerID string, schema *AddWalletCustomerSchema) (
	*AddWalletCustomerResponse, error,
) {
	req, err := c.NewRequest(
		http.MethodPost, c.c.API+"customers/v1/wallet",
		schema, map[string]string{"external_id": customerID},
	)
	if err != nil {
		return nil, err
	}
	resp := &AddWalletCustomerResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Returns customer details including payment methods, if saved.
func (c *Client) GetWalletCustomerPaymentInfo(customerID string) (*GetWalletInfoResponse, error) {
	req, err := c.NewRequest(
		http.MethodGet, c.c.API+"customers/v1/wallet",
		nil, map[string]string{"external_id": customerID},
	)
	if err != nil {
		return nil, err
	}
	resp := &GetWalletInfoResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Deletes customer payment method from wallet.
func (c *Client) DeleteWalletCustomerPayment(customerID string, schema *DeleteWalletCustomerSchema) (
	*DeleteWalletCustomerResponse, error,
) {
	req, err := c.NewRequest(
		http.MethodDelete, c.c.API+"customers/v1/wallet",
		schema, map[string]string{"external_id": customerID},
	)
	if err != nil {
		return nil, err
	}
	resp := &DeleteWalletCustomerResponse{}
	if err := c.Send(req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
