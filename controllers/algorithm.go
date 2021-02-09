package controllers

import (
	"math"
	"sort"
)

func MeanAndStd(x []float64) (mean float64, std float64) {
	sum, length := 0.0, float64(len(x))

	for _, v := range x {
		sum += v
	}

	mean = sum / length

	sum = 0.0

	for _, v := range x {
		sum += math.Pow((v - mean), 2.0)
	}

	std = math.Sqrt(sum / length)

	return
}

func Collect(prices []float64) []float64 {
	collectPoins := []float64{0.0, 0.2, 0.4, 0.5, 0.6, 0.8, 1.0}
	result := make([]float64, len(collectPoins))
	x := make([]float64, len(prices))
	length := float64(len(x) - 1)

	copy(x, prices)
	sort.Float64s(x)

	for i, v := range collectPoins {
		result[i] = x[int(length*v)]
	}

	return result
}
