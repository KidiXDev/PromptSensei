package index

type Progress struct {
	Stage   string
	Current int
	Total   int
	Detail  string
}

type ProgressFunc func(Progress)

func report(fn ProgressFunc, stage string, current int, total int, detail string) {
	if fn == nil {
		return
	}
	fn(Progress{
		Stage:   stage,
		Current: current,
		Total:   total,
		Detail:  detail,
	})
}
