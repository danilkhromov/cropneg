package main

import (
	"errors"
	"flag"
	"gocv.io/x/gocv"
	"gonum.org/v1/gonum/stat"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Threshold image with provided threshold value and ignore mask.
// Ignore mask is the outer bounds of scanned frame (film base combined with additional area surrounding it).
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

// Normalises coordinates of all contours in slice so they are not rotated more than -45 degrees.
func normaliseRectRotation(rawRects []gocv.RotatedRect) {

	for i, rect := range rawRects {
		if rect.Angle < -45 {
			contour := []image.Point{rect.Contour[1], rect.Contour[2], rect.Contour[3], rect.Contour[0]}
			rawRects[i].Contour = contour
			rawRects[i].Angle = rect.Angle + 90
		}
	}
}

// Finds median coordinates of all contours in slice.
// Returns rectangle which will be used to crop image.
func medianRect(rects []gocv.RotatedRect) (image.Rectangle, error) {

	var rect image.Rectangle

	if len(rects) == 0 {
		return rect, errors.New("rects is zero length")
	}

	normaliseRectRotation(rects)

	var zeroXs, zeroYs, oneXs, oneYs, twoXs, twoYs, threeXs, threeYs []float64

	for _, rect := range rects {
		zeroXs = append(zeroXs, float64(rect.Contour[0].X))
		zeroYs = append(zeroYs, float64(rect.Contour[0].Y))
		oneXs = append(oneXs, float64(rect.Contour[1].X))
		oneYs = append(oneYs, float64(rect.Contour[1].Y))
		twoXs = append(twoXs, float64(rect.Contour[2].X))
		twoYs = append(twoYs, float64(rect.Contour[2].Y))
		threeXs = append(threeXs, float64(rect.Contour[3].X))
		threeYs = append(threeYs, float64(rect.Contour[3].Y))
	}

	sort.Float64s(zeroXs)
	sort.Float64s(zeroYs)
	sort.Float64s(oneXs)
	sort.Float64s(oneYs)
	sort.Float64s(twoXs)
	sort.Float64s(twoYs)
	sort.Float64s(threeXs)
	sort.Float64s(threeYs)

	zeroX := stat.Quantile(0.5, stat.LinInterp, zeroXs, nil)
	zeroY := stat.Quantile(0.5, stat.LinInterp, zeroYs, nil)
	oneX := stat.Quantile(0.5, stat.LinInterp, oneXs, nil)
	oneY := stat.Quantile(0.5, stat.LinInterp, oneYs, nil)
	twoX := stat.Quantile(0.5, stat.LinInterp, twoXs, nil)
	twoY := stat.Quantile(0.5, stat.LinInterp, twoYs, nil)
	threeX := stat.Quantile(0.5, stat.LinInterp, threeXs, nil)
	threeY := stat.Quantile(0.5, stat.LinInterp, threeYs, nil)

	rect.Min = image.Pt(int(((oneX+zeroX)/2)*1.003), int(((oneY+twoY)/2)*1.003))
	rect.Max = image.Pt(int(((threeX+twoX)/2)*0.997), int(((threeY+zeroY)/2)*0.997))

	return rect, nil
}

// Finds exposure bounds of scanned image.
// Creates ignore mask for threshold method then gradually thresholds image
// and collects each contour that fits in between min and max capture area.
// Returns rectangle for crop calculated from median of all collected contours.
func findExposureBounds(img *gocv.Mat, wndw *gocv.Window, showDebugWindow bool) (image.Rectangle, error) {

	blGray := gocv.NewMat()
	gocv.BilateralFilter(*img, &blGray, 11, 17, 17)

	ignoreMask := gocv.NewMat()
	gocv.Threshold(blGray, &ignoreMask, 240, 255, gocv.ThresholdBinary)

	gocv.Dilate(
		ignoreMask,
		&ignoreMask,
		gocv.GetStructuringElement(gocv.MorphRect, image.Pt(5, 5)))

	gocv.GaussianBlur(ignoreMask, &ignoreMask, image.Pt(1, 1), 0, 0, gocv.BorderWrap)

	if showDebugWindow {
		wndw.IMShow(ignoreMask)
		wndw.WaitKey(1)
	}

	thrsh := gocv.NewMat()
	gocv.Threshold(blGray, &thrsh, 0, 255, gocv.ThresholdOtsu)

	unexpMask := gocv.NewMat()
	gocv.InRangeWithScalar(
		blGray,
		gocv.Scalar{},
		gocv.Scalar{Val1: 20, Val2: 20, Val3: 20},
		&unexpMask)

	if showDebugWindow {
		wndw.IMShow(unexpMask)
		wndw.WaitKey(0)
	}

	gocv.BitwiseOr(ignoreMask, unexpMask, &ignoreMask)
	gocv.BitwiseNot(ignoreMask, &ignoreMask)

	if showDebugWindow {
		wndw.IMShow(ignoreMask)
		wndw.WaitKey(0)
	}

	dims := img.Size()
	maxArea := (float64(dims[0]) * 0.92) * (float64(dims[1]) * 0.92)

	minCaptureArea := maxArea * 0.85

	var results []gocv.RotatedRect

	lThreshold := float32(240)
	for lThreshold > 0 {
		binary := thresholdImage(blGray, lThreshold, ignoreMask)

		debugImg := gocv.NewMat()
		gocv.CvtColor(binary, &debugImg, gocv.ColorGrayToBGR)

		rect, area := findLargestContourRect(&binary)

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

		if showDebugWindow {
			if rect.Contour != nil {
				rectPoints := gocv.NewMat()
				gocv.BoxPoints(rect, &rectPoints)

				var cntr [][]image.Point
				cntr = append(cntr, rect.Contour)
				gocv.DrawContours(&debugImg, cntr, -1, debugLineColour, 3)

				if showDebugWindow {
					wndw.IMShow(debugImg)
					wndw.WaitKey(1)
				}
			}
		}
	}

	return medianRect(results)
}

func main() {
	os.Exit(cropNegative())
}

func cropNegative() int {

	debug := flag.Bool("d", false, "show debug window")
	filename := flag.String("f", "", "scanned image to crop")
	targetFilename := flag.String("n", "", "filename with correct file extension of cropped image")

	flag.Parse()

	if *filename == "" {
		println("Please specify the scanned negative to process with command --file")
		return 1
	}

	window := gocv.NewWindow("analyze")
	img := gocv.IMRead(*filename, gocv.IMReadAnyColor)
	gray := gocv.IMRead(*filename, gocv.IMReadGrayScale)

	if *debug {
		window.ResizeWindow(800, 800)
		window.IMShow(img)
		window.WaitKey(0)
	}

	cropRect, err := findExposureBounds(&gray, window, *debug)

	if err != nil {
		log.Printf("error finding exposure bounds: %s", err)
		return 1
	}

	wbMask := gocv.NewMat()
	gocv.Threshold(gray, &wbMask, 253, 0, gocv.ThresholdToZero)
	gocv.BitwiseNot(wbMask, &wbMask)

	if *debug {
		gocv.Rectangle(&wbMask, cropRect, color.RGBA{}, 10)

		window.IMShow(wbMask)
		window.WaitKey(0)
	}

	var result bool
	if *targetFilename == "" {
		ext := filepath.Ext(*filename)
		result = gocv.IMWrite(strings.TrimSuffix(*filename, ext)+"_cropped"+ext, img.Region(cropRect))
	} else {
		result = gocv.IMWrite(*targetFilename, img.Region(cropRect))
	}


	if result {
		return 0
	}

	return 1
}
