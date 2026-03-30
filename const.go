package aero

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
	mCount   = 7
)

type methodBit uint16

const (
	methodBitGET    methodBit = 1 << 0
	methodBitPOST   methodBit = 1 << 1
	methodBitPUT    methodBit = 1 << 2
	methodBitPATCH  methodBit = 1 << 3
	methodBitDELETE methodBit = 1 << 4
	methodBitHEAD   methodBit = 1 << 5
)

var methodBits = [mCount]methodBit{
	mGET:    methodBitGET,
	mPOST:   methodBitPOST,
	mPUT:    methodBitPUT,
	mPATCH:  methodBitPATCH,
	mDELETE: methodBitDELETE,
	mHEAD:   methodBitHEAD,
}

func methodIndex(m string) int {
	switch m {
	case "GET":
		return mGET
	case "POST":
		return mPOST
	case "PUT":
		return mPUT
	case "PATCH":
		return mPATCH
	case "DELETE":
		return mDELETE
	case "HEAD":
		return mHEAD
	case "OPTIONS":
		return mOPTIONS
	}

	return -1
}

func methodString(mi int) string {
	switch mi {
	case mGET:
		return "GET"
	case mPOST:
		return "POST"
	case mPUT:
		return "PUT"
	case mPATCH:
		return "PATCH"
	case mDELETE:
		return "DELETE"
	case mHEAD:
		return "HEAD"
	case mOPTIONS:
		return "OPTIONS"
	}

	return ""
}
