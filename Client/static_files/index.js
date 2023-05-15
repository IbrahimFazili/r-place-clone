const USERNAME_KEY = 'username'
/**
 * @type {Board}
 */
var board

/**
 * @type {Renderer}
 */
var renderer

/**
 * @param {Object} stats 
 */
function LogStats(stats) {
	// console.clear()
	for (const key in stats) {
		console.log(key, stats[key])
	}
}

/**
 * @class
 * @public
 */
class Board {
	colorMapping = {
		0: "#ffffff",
		1: "#000000",
		2: "#2450a4",
		3: "#00a368",
		4: "#be0039",
		5: "#ffa800",
		6: "#FFFF00",
		7: "#6d482f",
		8: "#811e9f",
		9: "#b44ac0",
		10: "#7eed56",
		11: "#3690ea",
		12: "#be0049",
		13: "#ffd631",
		14: "#6d001a",
		15: "#e4abff",
	};
	dimension = 0;

	/**
	 * @type {Array}
	 * @public
	 */
	pixels = null;


	/**
	 * @private
	 */
	pixelColourToFill = "";

	/**
	 * @type {HTMLCanvasElement}
	 * @public
	 */
	canvas = null;

	/**
	 * @param {number} dimension board dimension - creates a board of dimension x dimension size 
	 * @param {HTMLCanvasElement} canvas canvas that this board will render to
	 */
	constructor(dimension, canvas) {
		this.canvas = canvas;
		this.dimension = dimension;
		// this.pixels = new Uint8ClampedArray(dimension * dimension * 4)
		this.pixels = new Array(this.dimension * this.dimension)

		for (let i = 0; i < this.pixels.length; i++) {
			// this.pixels[i] = 255
			this.pixels[i] = "#FFFFFF"
		}
	}

	/**
	 * render the board onto a canvas
	 * @param {HTMLCanvasElement} canvas the canvas to render the board to 
	 */
	render() {
		if (!this.canvas)
			return;

		const ctx = this.canvas.getContext("2d")
		ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height)
		ctx.putImageData(new ImageData(this.pixels, this.dimension, this.dimension), 0, 0)
	}

	getMousePos(canvas, evt) {
		var rect = canvas.getBoundingClientRect() // abs. size of element

		return {
			x: Math.floor((evt.clientX - rect.left) / PIXEL_SCALE),   // scale mouse coordinates after they have
			y: Math.floor((evt.clientY - rect.top) / PIXEL_SCALE)     // been adjusted to be relative to element
		}
	}

	/**
	 * writes a single pixel to the server
	 * @param {HTMLCanvasElement} canvas the canvas to render to
	 * @param {MouseEvent} event
	*/
	writePixel(canvas, event) {
		const ctx = canvas.getContext("2d")

		const rect = canvas.getBoundingClientRect()
		const { x, y } = this.getMousePos(canvas, event)
		if (x < 0 || x >= this.dimension || y < 0 || y >= this.dimension) {
			return
		}

		console.log("x: " + x + " y: " + y)

		// @TODO: chcek if 5 mins is over
		// Push update using /api/writepixel

		const color = this.getKeyByValue(this.colorMapping, this.pixelColourToFill)
		if (color != null || color != undefined)
		{
			API_WritePixel(x, y, color, localStorage.getItem(USERNAME_KEY))
		}
	}

	/**
	 * @private
	 */
	generateRandomUser()
	{
		let username = "user"
		for (let i = 0; i < 5; i++)
		{			
			username += Math.trunc(Math.random() * 10).toString()
		}

		return username
	}

	/**
	 * this function is used to receive the board from the bitefield and update it locally
	*/
	downloadLatestBoard() {
		FetchBoard(`${GetEndpoint()}/api/board`, (pixels, err) => {
			if (err) {
				console.error("FetchBoard: ", err)
				return
			}

			const encoder = new TextEncoder("utf-8")
			const buf = new DataView(new ArrayBuffer(pixels.length))
			for (let i = 0; i < pixels.length; i++) {
				buf.setUint8(i, encoder.encode(pixels[i])[0])
			}

			const responseArray = new Uint8Array(buf.buffer)
			
			// now iterate and get the low and high and put in new array
			var bytesArray = []
			responseArray.forEach((pixelByte, _) => {
				var firstPixel = pixelByte >> 4
				var secondPixel = pixelByte & 0x0F
				// find each of their respective colour
				var firstPixelColour = this.getColorMapping()[firstPixel]
				var secondPixelColour = this.getColorMapping()[secondPixel]

				bytesArray.push(firstPixelColour)
				bytesArray.push(secondPixelColour)
			})

			this.UpdateFullBoard(bytesArray)
		})
	}

	/**
	 * @private
	 * @param {Uint8ClampedArray} pixels
	 */
	UpdateFullBoard(pixels) {
		this.pixels = pixels
		renderer.draw()
	}

	/**
	 * 
	 * @param {MessageEvent} ev 
	 */
	HandlePixelUpdate(ev) {
		const update = JSON.parse(ev.data)
		const hexColor = this.colorMapping[update.color]
		this.pixels[(this.dimension * update.y) + update.x] = hexColor

		renderer.DrawSinglePixel(update.x, update.y)
	}

	convertToRgb(colour) {
		var aRgbHex = colour.match(/.{1,2}/g);
		var aRgb = [
			parseInt(aRgbHex[0], 16),
			parseInt(aRgbHex[1], 16),
			parseInt(aRgbHex[2], 16),
		];
		return aRgb;
	}

	setSelectedColor(e) {
		this.pixelColourToFill = e;
	}

	getKeyByValue(object, value) {
		return Object.keys(object).find((key) => object[key] === value);
	}

	getColorMapping() {
		return this.colorMapping;
	}

	getDimension() {
		return this.dimension
	}
}

function CreatePalette() {
	// render the button for palette
	var palette = document.getElementById("palette");
	var colorMapping = board.getColorMapping()

	Object.keys(colorMapping).forEach((key) => {
		var button = document.createElement("button");
		button.classList.add("palette-colour");
		button.style.backgroundColor = colorMapping[key];
		button.addEventListener("click", function () {
			board.setSelectedColor(colorMapping[key]);
		});
		palette.appendChild(button);
	})
}

window.onload = async function () {
	const canvas = document.getElementById("grid");
	board = new Board(1000, canvas);
	
	let username = prompt("Please enter your name")
	localStorage.setItem(USERNAME_KEY, username)

	await InitAPIConnection((ev) => board.HandlePixelUpdate(ev))

	board.downloadLatestBoard();
	setInterval(() => {
		console.log("Downloading full snapshot at ", Date.now())
		board.downloadLatestBoard()
	}, 5000)
	CreatePalette();
	const zoomOut = document.getElementById("zoomOut");
	zoomOut.addEventListener('click', () => {
		adjustZoom(100 * -SCROLL_SENSITIVITY)
	})

	const zoomIn = document.getElementById("zoomIn");
	zoomIn.addEventListener('click', () => {
		adjustZoom(-100 * -SCROLL_SENSITIVITY)
	})

	canvas.addEventListener("mouseup", (e) => board.writePixel(canvas, e));

	renderer = new Renderer(canvas)
};
