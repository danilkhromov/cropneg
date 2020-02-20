package main

import (
	"flag"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"image"
	"image/color"
	"sort"
)

func thresholdImage(img gocv.Mat, lThreshold float32, ignoreMask gocv.Mat) gocv.Mat {

	binary := gocv.NewMat()
	gocv.Threshold(img, &binary, lThreshold, 255, gocv.ThresholdBinary)

	gocv.BitwiseAnd(ignoreMask, binary, &binary)

	gocv.Dilate(
		binary,
		&binary,
		gocv.GetStructuringElement(gocv.MorphRect, image.Pt(10, 10)))
	gocv.Erode(
		binary,
		&binary,
		gocv.GetStructuringElement(gocv.MorphRect, image.Pt(10, 10)))

	return binary
}

func findLargestContourRect(binary *gocv.Mat) (gocv.RotatedRect, float64) {

	var largestRect gocv.RotatedRect
	var largestArea = float64(0)

	contours := gocv.FindContours(*binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	for _, cnt := range contours {
		area := gocv.ContourArea(cnt)

		if area > largestArea {
			largestArea = area
			largestRect = gocv.MinAreaRect(cnt)
		}
	}

	return largestRect, largestArea
}

func normaliseRectRotation(rawRects []gocv.RotatedRect) {

	var rects []gocv.RotatedRect

	for _, rect := range rawRects {

		if rect.Angle < -45 {
			rect.Angle = rect.Angle + 90
		}

		rects = append(rects, rect)
	}
}

func medianRect(rects []gocv.RotatedRect) gocv.RotatedRect {

	normaliseRectRotation(rects)

	var zeroXs, zeroYs, oneXs, oneYs, twoXs, twoYs, threeXs, threeYs []float64
	var minBXs, minBYs []float64
	var maxBXs, maxBys []float64
	var cntrXs, cntrYs []float64
	var widths, heights []float64
	var angles []float64

	for _, rect := range rects {
		zeroXs = append(zeroXs, float64(rect.Contour[0].X))
		zeroYs = append(zeroYs, float64(rect.Contour[0].Y))
		oneXs = append(oneXs, float64(rect.Contour[1].X))
		oneYs = append(oneYs, float64(rect.Contour[1].Y))
		twoXs = append(twoXs, float64(rect.Contour[2].X))
		twoYs = append(twoYs, float64(rect.Contour[2].Y))
		threeXs = append(threeXs, float64(rect.Contour[3].X))
		threeYs = append(threeYs, float64(rect.Contour[3].Y))
		minBXs = append(minBXs, float64(rect.BoundingRect.Min.X))
		minBYs = append(minBYs, float64(rect.BoundingRect.Min.Y))
		maxBXs = append(maxBXs, float64(rect.BoundingRect.Max.X))
		maxBys = append(maxBys, float64(rect.BoundingRect.Max.Y))
		cntrXs = append(cntrXs, float64(rect.Center.X))
		cntrYs = append(cntrYs, float64(rect.Center.Y))
		widths = append(widths, float64(rect.Width))
		heights = append(heights, float64(rect.Height))
		angles = append(angles, rect.Angle)
	}

	sort.Float64s(zeroXs)
	sort.Float64s(zeroYs)
	sort.Float64s(oneXs)
	sort.Float64s(oneYs)
	sort.Float64s(twoXs)
	sort.Float64s(twoYs)
	sort.Float64s(threeXs)
	sort.Float64s(threeYs)
	sort.Float64s(minBXs)
	sort.Float64s(minBYs)
	sort.Float64s(maxBXs)
	sort.Float64s(maxBys)
	sort.Float64s(cntrXs)
	sort.Float64s(cntrYs)
	sort.Float64s(widths)
	sort.Float64s(heights)
	sort.Float64s(angles)

	return gocv.RotatedRect{
		Contour: []image.Point{
			image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, zeroXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, zeroYs, nil))),
			image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, oneXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, oneYs, nil))),
			image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, twoXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, twoYs, nil))),
			image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, threeXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, threeYs, nil))),
		},
		BoundingRect: image.Rectangle{
			Min: image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, minBXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, minBYs, nil))),
			Max: image.Pt(
				int(stat.Quantile(0.5, stat.LinInterp, maxBXs, nil)),
				int(stat.Quantile(0.5, stat.LinInterp, maxBys, nil))),
		},
		Center: image.Pt(
			int(stat.Quantile(0.5, stat.LinInterp, cntrXs, nil)),
			int(stat.Quantile(0.5, stat.LinInterp, cntrYs, nil))),
		Width:  int(stat.Quantile(0.5, stat.LinInterp, widths, nil)),
		Height: int(stat.Quantile(0.5, stat.LinInterp, heights, nil)),
		Angle:  stat.Quantile(0.5, stat.LinInterp, angles, nil),
	}
}

func findExposureBounds(wndw *gocv.Window, img *gocv.Mat, showOutputWindow bool) gocv.RotatedRect {

	gray := gocv.NewMat()
	gocv.CvtColor(*img, &gray, gocv.ColorBGRToGray)

	blGray := gocv.NewMat()
	gocv.BilateralFilter(gray, &blGray, 11, 17, 17)

	ignoreMask := gocv.NewMat()
	gocv.Threshold(blGray, &ignoreMask, 240, 255, gocv.ThresholdBinary)

	gocv.Dilate(
		ignoreMask,
		&ignoreMask,
		gocv.GetStructuringElement(gocv.MorphShape(0), image.Pt(15, 15)))

	hsv := gocv.NewMat()
	gocv.CvtColor(*img, &hsv, gocv.ColorBGRToGray)
	gocv.Threshold(blGray, &hsv, 0, 255, gocv.ThresholdOtsu)

	unexpMask := gocv.NewMat()
	gocv.InRangeWithScalar(
		blGray,
		gocv.Scalar{},
		gocv.Scalar{Val1: 20, Val2: 20, Val3: 20},
		&unexpMask)

	gocv.BitwiseOr(ignoreMask, unexpMask, &ignoreMask)
	gocv.BitwiseNot(ignoreMask, &ignoreMask)

	wndw.IMShow(ignoreMask)
	wndw.WaitKey(0)

	dims := img.Size()
	maxArea := (float64(dims[0]) * 0.98) * (float64(dims[1]) * 0.98)

	minCaptureArea := maxArea * 0.65

	algos := []func(binary *gocv.Mat) (gocv.RotatedRect, float64){findLargestContourRect}

	var results []gocv.RotatedRect

	for _, fun := range algos {
		lThreshold := float32(240)

		for lThreshold > 0 {
			binary := thresholdImage(blGray, lThreshold, ignoreMask)

			debugImg := gocv.NewMat()
			gocv.CvtColor(binary, &debugImg, gocv.ColorGrayToBGR)

			rect, area := fun(&binary)

			if area >= maxArea {
				break
			}

			var debugLineColour color.RGBA
			if area >= minCaptureArea {
				results = append(results, rect)
				lThreshold -= 5

				debugLineColour = color.RGBA{G: 255}
			} else {
				lThreshold -= 5

				debugLineColour = color.RGBA{R: 255}
			}

			if showOutputWindow {
				if rect.Contour != nil {
					rectPoints := gocv.NewMat()
					gocv.BoxPoints(rect, &rectPoints)

					var cntr [][]image.Point
					cntr = append(cntr, rect.Contour)
					gocv.DrawContours(&debugImg, cntr, -1, debugLineColour, 3)

					wndw.IMShow(debugImg)
					wndw.WaitKey(1)
				}
			}
		}
	}

	return medianRect(results)
}

func main() {

	filename := flag.String("file", "", "scan to analyze")

	flag.Parse()

	if *filename == "" {
		println("Please specify the scanned negative to process with command --file")
		return
	}

	window := gocv.NewWindow("analyze")
	img := gocv.IMRead(*filename, gocv.IMReadAnyColor)

	window.ResizeWindow(500, 500)
	window.IMShow(img)
	window.WaitKey(0)

	gray := gocv.NewMat()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	rawRect := findExposureBounds(window, &img, true)

	var cntr [][]image.Point
	cntr = append(cntr, rawRect.Contour)

	wbMask := gocv.NewMat()
	gocv.CvtColor(img, &wbMask, gocv.ColorBGRToGray)
	gocv.Threshold(wbMask, &wbMask, 253, 0, gocv.ThresholdToZero)
	gocv.BitwiseNot(wbMask, &wbMask)

	gocv.FillPoly(&wbMask, cntr, color.RGBA{})

	window.IMShow(wbMask)
	window.WaitKey(0)

	gocv.DrawContours(&img, cntr, -1, color.RGBA{G: 255}, 3)

	window.IMShow(img)
	window.WaitKey(0)

	gocv.IMWrite(*filename + "_analyzed.jpg", img)
}
