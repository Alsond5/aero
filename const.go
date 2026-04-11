package aero

import "net/http"

const (
	HeaderTransferEncoding    = "Transfer-Encoding"
	HeaderContentType         = "Content-Type"
	HeaderContentLength       = "Content-Length"
	HeaderLink                = "Link"
	HeaderContentDisposition  = "Content-Disposition"
	HeaderXContentTypeOptions = "X-Content-Type-Options"
	HeaderReferer             = "Referer"
	HeaderLocation            = "Location"
	HeaderVary                = "Vary"
	HeaderCacheControl        = "Cache-Control"
	HeaderConnection          = "Connection"
)

const (
	MIMEApplicationJSON = "application/json"
	MIMEApplicationXML  = "application/xml"
	MIMETextXML         = "text/xml"
	MIMEApplicationForm = "application/x-www-form-urlencoded"
	MIMEMultipartForm   = "multipart/form-data"
)

const (
	maxParamCount    = 8
	maxParamCountStr = "8"
)

const (
	mGET     = 0
	mPOST    = 1
	mPUT     = 2
	mPATCH   = 3
	mDELETE  = 4
	mHEAD    = 5
	mOPTIONS = 6
	mTRACE   = 7
	mCount   = 8
)

type methodBit uint16

const (
	methodBitGET    methodBit = 1 << 0
	methodBitPOST   methodBit = 1 << 1
	methodBitPUT    methodBit = 1 << 2
	methodBitPATCH  methodBit = 1 << 3
	methodBitDELETE methodBit = 1 << 4
	methodBitHEAD   methodBit = 1 << 5
	methodBitTRACE  methodBit = 1 << 6
)

var methodBits = [mCount]methodBit{
	mGET:    methodBitGET,
	mPOST:   methodBitPOST,
	mPUT:    methodBitPUT,
	mPATCH:  methodBitPATCH,
	mDELETE: methodBitDELETE,
	mHEAD:   methodBitHEAD,
	mTRACE:  methodBitTRACE,
}

func methodIndex(m string) int {
	switch m {
	case http.MethodGet:
		return mGET
	case http.MethodPost:
		return mPOST
	case http.MethodPut:
		return mPUT
	case http.MethodPatch:
		return mPATCH
	case http.MethodDelete:
		return mDELETE
	case http.MethodHead:
		return mHEAD
	case http.MethodOptions:
		return mOPTIONS
	case http.MethodTrace:
		return mTRACE
	}

	return -1
}

func methodString(mi int) string {
	switch mi {
	case mGET:
		return http.MethodGet
	case mPOST:
		return http.MethodPost
	case mPUT:
		return http.MethodPut
	case mPATCH:
		return http.MethodPatch
	case mDELETE:
		return http.MethodDelete
	case mHEAD:
		return http.MethodHead
	case mOPTIONS:
		return http.MethodOptions
	case mTRACE:
		return http.MethodTrace
	}

	return ""
}
