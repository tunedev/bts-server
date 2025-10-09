package email

import (
	"bytes"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"

	"github.com/skip2/go-qrcode"
)

//go:embed templates/*.html
var templateFS embed.FS

type SendRSVPConfirmedParam struct {
	GuestName      string
	NumberOfGuests int
	RSVPID         string
	Phone          string
}

// SendRSVPConfirmed sends the confirmation email with a unique QR code.
func (m Mailer) SendRSVPConfirmed(to string, param SendRSVPConfirmedParam) error {
	subject := "Your RSVP is Confirmed - See you there!"

	qrData := fmt.Sprintf(`{"rsvpID":"%s","guestName":"%s","phone":"%s"}`, param.RSVPID, param.GuestName, param.Phone)

	var png []byte
	png, err := qrcode.Encode(qrData, qrcode.Medium, 256)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}

	qrCodeBase64 := base64.StdEncoding.EncodeToString(png)
	qrCodeDataURL := "data:image/png;base64," + qrCodeBase64

	data := struct {
		GuestName      string
		NumberOfGuests int
		QRCode         template.URL
		Phone          string
	}{
		GuestName:      param.GuestName,
		NumberOfGuests: param.NumberOfGuests,
		QRCode:         template.URL(qrCodeDataURL),
		Phone:          param.Phone,
	}

	// Parse the specific content template first
	contentTmpl, err := template.New("rsvp_confirmed.html").ParseFS(templateFS, "templates/rsvp_confirmed.html")
	if err != nil {
		return err
	}

	var contentBody bytes.Buffer
	if err := contentTmpl.Execute(&contentBody, data); err != nil {
		return err
	}

	// Now parse the main layout and inject the content
	layoutData := struct {
		Body             template.HTML
		ShowLocationLink bool
	}{
		Body:             template.HTML(contentBody.String()),
		ShowLocationLink: true,
	}

	layoutTmpl, err := template.New("layout.html").ParseFS(templateFS, "templates/layout.html")
	if err != nil {
		return err
	}

	var finalBody bytes.Buffer
	if err := layoutTmpl.Execute(&finalBody, layoutData); err != nil {
		return err
	}

	// Send the final, assembled email
	return m.Send(to, subject, finalBody.String())
}

// SendLoginOTP sends the one-time password for admin login using the main layout.
func (m Mailer) SendLoginOTP(to, otp string) error {
	subject := "Your Sign-In Code for BTS Wedding Admin"
	data := struct {
		OTP string
	}{OTP: otp}

	// The 'body' here is the final, fully-rendered HTML
	body, err := m.parseLayout("otp.html", data)
	if err != nil {
		return err
	}
	return m.Send(to, subject, body)
}

// SendRSVPReceived notifies a guest that their RSVP is pending, using the main layout.
func (m Mailer) SendRSVPReceived(to, guestName string) error {
	subject := "We've Received Your RSVP!"
	data := struct {
		GuestName string
	}{GuestName: guestName}

	body, err := m.parseLayout("rsvp_pending.html", data)
	if err != nil {
		return err
	}
	return m.Send(to, subject, body)
}

// SendRSVPRejected notifies a guest that their RSVP was rejected, using the main layout.
func (m Mailer) SendRSVPRejected(to, guestName string) error {
	subject := "An Update on Your RSVP"
	data := struct {
		GuestName        string
		ShowLocationLink bool
	}{GuestName: guestName}

	body, err := m.parseLayout("rsvp_rejected.html", data)
	if err != nil {
		return err
	}
	return m.Send(to, subject, body)
}

// parseLayout is the new helper function that injects content into the main layout.
func (m Mailer) parseLayout(contentFile string, data interface{}) (string, error) {
	contentTmpl, err := template.New(contentFile).ParseFS(templateFS, "templates/"+contentFile)
	if err != nil {
		return "", err
	}

	var contentBody bytes.Buffer
	if err := contentTmpl.Execute(&contentBody, data); err != nil {
		return "", err
	}

	layoutTmpl, err := template.New("layout.html").ParseFS(templateFS, "templates/layout.html")
	if err != nil {
		return "", err
	}

	layoutData := struct {
		Body             template.HTML
		ShowLocationLink bool
	}{
		Body: template.HTML(contentBody.String()),
	}

	var finalBody bytes.Buffer
	if err := layoutTmpl.Execute(&finalBody, layoutData); err != nil {
		return "", err
	}

	return finalBody.String(), nil
}
