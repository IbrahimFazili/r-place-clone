let cameraOffset = { x: window.innerWidth / 2, y: window.innerHeight / 2 }
let cameraZoom = 1
let MAX_ZOOM = 15
let MIN_ZOOM = 0.1
let SCROLL_SENSITIVITY = 0.01
let PIXEL_SCALE = 10
/**
 * @type {CanvasRenderingContext2D}
 */
let ctx = null

function Clamp(n, lower, upper)
{
    if (n < lower)
    {
        n = lower
    }
    else if (n > upper)
    {
        n = upper
    }

    return n
}

// Gets the relevant location from a mouse or single touch event
function getEventLocation(e) {
    if (e.touches && e.touches.length == 1) {
        return { x: e.touches[0].clientX, y: e.touches[0].clientY }
    }
    else if (e.clientX && e.clientY) {
        return { x: e.clientX, y: e.clientY }
    }
}

/**
 * 
 * @param {number} x 
 * @param {number} y 
 * @param {number} width 
 * @param {number} height 
 * @param {CanvasRenderingContext2D} ctx 
 */
function drawRect(x, y, width, height, ctx) {
    ctx.fillRect(x, y, width, height)
    ctx.lineWidth = 0.5
    ctx.strokeStyle = 'lightgray'
    ctx.strokeRect(x, y, width, height)
}

function drawText(text, x, y, size, font) {
    ctx.font = `${size}px ${font}`
    ctx.fillText(text, x, y)
}

let isDragging = false
let dragStart = { x: 0, y: 0 }

function onPointerDown(e) {
    isDragging = true
    dragStart.x = getEventLocation(e).x / cameraZoom - cameraOffset.x
    dragStart.y = getEventLocation(e).y / cameraZoom - cameraOffset.y
}

function onPointerUp(e) {
    isDragging = false
    initialPinchDistance = null
    lastZoom = cameraZoom
}

function onPointerMove(e) {
    if (isDragging) {
        cameraOffset.x = getEventLocation(e).x / cameraZoom - dragStart.x
        cameraOffset.y = getEventLocation(e).y / cameraZoom - dragStart.y
    }
}

function handleTouch(e, singleTouchHandler) {
    if (e.touches.length == 1) {
        singleTouchHandler(e)
    }
    else if (e.type == "touchmove" && e.touches.length == 2) {
        isDragging = false
        handlePinch(e)
    }
}

let initialPinchDistance = null
let lastZoom = cameraZoom

function handlePinch(e) {
    e.preventDefault()

    let touch1 = { x: e.touches[0].clientX, y: e.touches[0].clientY }
    let touch2 = { x: e.touches[1].clientX, y: e.touches[1].clientY }

    // This is distance squared, but no need for an expensive sqrt as it's only used in ratio
    let currentDistance = (touch1.x - touch2.x) ** 2 + (touch1.y - touch2.y) ** 2

    if (initialPinchDistance == null) {
        initialPinchDistance = currentDistance
    }
    else {
        adjustZoom(null, currentDistance / initialPinchDistance)
    }
}

function adjustZoom(zoomAmount) {
    PIXEL_SCALE = Clamp(PIXEL_SCALE + zoomAmount, MIN_ZOOM, MAX_ZOOM)
    renderer.draw()
}

class Renderer {
    /**
     * @type {HTMLCanvasElement}
     * @readonly
     */
    outputCanvas

    /**
     * @type {HTMLCanvasElement}
     * @private
     */
    scratchCanvas

    constructor(canvas) {
        this.outputCanvas = canvas
        this.scratchCanvas = document.createElement('canvas')
        this.scratchCanvas.hidden = true
        this.scratchCanvas.width = 1000
        this.scratchCanvas.height = 1000

        ctx = canvas.getContext("2d")
        ctx.imageSmoothingEnabled = false

        this.RegisterEventListeners()
    }

    Start() {
        // requestAnimationFrame(() => this.draw())
        setInterval(() => this.draw(), 1000)
    }

    RegisterEventListeners() {
        const canvas = this.outputCanvas
        canvas.addEventListener('mousedown', onPointerDown)
        canvas.addEventListener('touchstart', (e) => handleTouch(e, onPointerDown))
        canvas.addEventListener('mouseup', onPointerUp)
        canvas.addEventListener('touchend', (e) => handleTouch(e, onPointerUp))
        canvas.addEventListener('mousemove', onPointerMove)
        canvas.addEventListener('touchmove', (e) => handleTouch(e, onPointerMove))
    }

    DrawSinglePixel(x, y) {
        const ctx = this.outputCanvas.getContext("2d")
        const pixelSize = PIXEL_SCALE

        let drawX = x * pixelSize, drawY = y * pixelSize
        const offset = (y * board.dimension) + x
        ctx.fillStyle = board.pixels[offset]
        drawRect(drawX, drawY, pixelSize, pixelSize, ctx)
    }

    SlowDraw() {
        const ctx = this.outputCanvas.getContext("2d")
        const pixelSize = PIXEL_SCALE
        let drawX = 0, drawY = 0
        for (let i = 0; i < board.pixels.length; i++){
            if (i > 0 && (i % board.dimension == 0)) {
                drawX = 0
                drawY += pixelSize
            }

            ctx.fillStyle = board.pixels[i]
            drawRect(drawX, drawY, pixelSize, pixelSize, ctx)
            drawX += pixelSize
        }
    }

    scaleImageData(imageData, scale, ctx) {
        var scaled = ctx.createImageData(imageData.width * scale, imageData.height * scale);
        var subLine = ctx.createImageData(scale, 1).data
        for (var row = 0; row < imageData.height; row++) {
            for (var col = 0; col < imageData.width; col++) {
                var sourcePixel = imageData.data.subarray(
                    (row * imageData.width + col) * 4,
                    (row * imageData.width + col) * 4 + 4
                );
                for (var x = 0; x < scale; x++) subLine.set(sourcePixel, x*4)
                for (var y = 0; y < scale; y++) {
                    var destRow = row * scale + y;
                    var destCol = col * scale;
                    scaled.data.set(subLine, (destRow * scaled.width + destCol) * 4)
                }
            }
        }
    
        return scaled;
    }

    draw() {
        const scratchCtx = this.scratchCanvas.getContext("2d")
        const canvas = this.outputCanvas
        canvas.width = board.dimension * PIXEL_SCALE
        canvas.height = board.dimension * PIXEL_SCALE

        // Translate to the canvas centre before zooming - so you'll always zoom on what you're looking directly at
        // ctx.translate(window.innerWidth / 2, window.innerHeight / 2)
        // ctx.scale(cameraZoom, cameraZoom)
        // ctx.translate(-window.innerWidth / 2 + cameraOffset.x, -window.innerHeight / 2 + cameraOffset.y)

        ctx.clearRect(0, 0, window.innerWidth, window.innerHeight)
        scratchCtx.clearRect(0, 0, window.innerWidth, window.innerHeight)
        this.SlowDraw()

        // const scaledImage = this.scaleImageData(new ImageData(board.pixels, board.dimension, board.dimension), scaleFactor, ctx)
        // ctx.putImageData(scaledImage, 0, 0)
        // ctx.drawImage(this.scratchCanvas, 0, 0)

        // requestAnimationFrame(() => this.draw())
    }
}
