package main

import (
	"math"
)

func dewpointFahrenheit(t, rh float32) float32 {
	return cToF(dewpointCelcius(fToC(t), rh))
}

// dewpointCelcius calculates the dewpoint in °C using the Magnus formula
// per https://en.wikipedia.org/wiki/Dew_point#Calculating_the_dew_point
func dewpointCelcius(t, rh float32) float32 {
	if t < 0 {
		return 0.0
	}

	// Prevent a divide-by-zero
	if t == -237.7 {
		return 0.0
	}

	rh = rh / 100.0
	γ := 17.27*t/(237.7+t) + float32(math.Log(float64(rh)))

	// Prevent a divide-by-zero
	if γ == 17.27 {
		return 0.0
	}

	TdpC := 237.7 * γ / (17.27 - γ)

	return TdpC
}

// windchillFahrenheit calculates the wind chill using a calculation from
// http://www.nws.noaa.gov/om/winter/windchill.shtml
func windchillFahrenheit(t, ws float32) float32 {
	// Wind chill is only valid for temps less than or equal to 50°F and wind speeds over 0 MPH.
	if t >= 50 || ws <= 0 {
		return t
	}
	WcF := 35.74 + 0.6215*t + (-35.75+0.4275*t)*float32(math.Pow(float64(ws), 0.16))
	return WcF
}

// heatIndex calculates the heat index using the calculation from
// http://www.wpc.ncep.noaa.gov/html/heatindex_equation.shtml
func heatIndexFahrenheit(t, rh float32) float32 {
	// Heat index is only valid for temps over 80°F and relative humidity over 40%
	if t < 80.0 || rh <= 40.0 {
		return t
	}

	heatIdx := -42.379 + 2.04901523*t + 10.14333127*rh - 0.22475541*t*rh - 6.83783e-3*t*t - 5.481717e-2*rh*rh + 1.22874e-3*t*t*rh + 8.5282e-4*t*rh*rh - 1.99e-6*t*t*rh*rh
	return heatIdx
}

func fToC(t float32) float32 {
	return ((t - 32.0) * 5.0 / 9.0)
}

func cToF(t float32) float32 {
	return (t * 9.0 / 5.0) + 32.0
}
