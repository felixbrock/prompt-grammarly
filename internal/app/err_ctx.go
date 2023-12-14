package app

type errCtx struct {
	Code  int
	Title string
	Msg   string
}

func get400() errCtx {
	return errCtx{
		Code:  400,
		Title: "Bad request",
		Msg:   "Sorry, we couldn't find the page you were looking for.",
	}
}

func get405() errCtx {
	return errCtx{
		Code:  405,
		Title: "Method not allowed",
		Msg:   "Sorry, we couldn't find the page you were looking for.",
	}
}

func get500() errCtx {
	return errCtx{
		Code:  500,
		Title: "Internal server error",
		Msg:   "Sorry, there was an internal server error.",
	}
}
