package main

type filterFunc func(i int) bool

func filter(src []string, fn ...filterFunc) []string {
	if len(fn) == 0 {
		return nil
	}

	results := make([]string, 0, len(src))
	for i := 0; i < len(src); i++ {
		allok := true
		for _, f := range fn {
			if !f(i) {
				allok = false
			}
		}
		if allok {
			results = append(results, src[i])
		}
	}
	return results
}
