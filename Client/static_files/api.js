const SOCKET_ENDPOINT = "a3-te-loadb-1o7xofs5mgr87-1713514427.us-east-1.elb.amazonaws.com"
const ENDPOINT = "d238qs7496gem2.cloudfront.net"
const PROXY = "localhost:8000"
const USE_PROXY = false
/**
 * @type {WebSocket}
 */
var socket
var GoCtx

/**
 * This callback type is called `BoardFetchFinishCallback` and is displayed as a global symbol.
 *
 * @callback BoardFetchFinishCallback
 * @param {string} pixels
 * @param {error} err
 */

/**
 * This callback type is called `PixelUpdateCallback` and is displayed as a global symbol.
 *
 * @callback PixelUpdateCallback
 * @param {MessageEvent<any>} ev Socket update event
 */

/**
 * @param {BoardFetchFinishCallback} ondone
 * @returns {null}

*/
function FetchBoard(ondone) { }

/**
 * @param {MessageEvent<any>} ev
 * @returns {null}

*/
function HandleMessage(ev) {
    console.log("Event: ", ev)
    console.log("Raw data: ", ev.data)
    console.log("Parsed: ", JSON.parse(ev.data))
}

function GetEndpoint() {
    if (USE_PROXY)
        return `http://${PROXY}/http://${ENDPOINT}`
    
    return `http://${ENDPOINT}`
}

/**
 * 
 * @param {number} x x coordinate of pixel from top left
 * @param {number} y y coordinate of pixel from top left
 * @param {string} user username of user writing
 * @param {number} color a valid color number from color map
 */
async function API_WritePixel(x, y, color, user) {
    console.log({
        "X": x,
        "Y": y,
        "Col": parseInt(color),
        "User": user
    })
    const waitTime = await API_GetUser(user)
    if (waitTime > 0) {
        alert(`You still have to wait ${waitTime} seconds before putting another tile.`)
        return
    }

    const finalURL = USE_PROXY ? `http://${PROXY}/http://${ENDPOINT}/api/writepixel` : `http://${ENDPOINT}/api/writepixel`

    const res = await fetch(finalURL, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            "X": x,
            "Y": y,
            "Col": parseInt(color),
            "User": user
        })
    })
    if (res.status != 200) {
        console.error("[API]: /api/writepixel failed due to", res)
    }
}

async function API_GetUser(user){

    const finalURL = USE_PROXY ? `http://${PROXY}/http://${ENDPOINT}/api/getuser` : `http://${ENDPOINT}/api/getuser`

    const waitTime = await fetch(finalURL, {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            "User": user
        })
    }).then((response) => {return response.json()})
    return waitTime
}
/**
 * @param {PixelUpdateCallback} onupdate This callback is called when the socket server pushes an update for a single pixel
 */
async function InitAPIConnection(onupdate) {
    // golang execution context
    const go = new Go();

    const wasmCtx = await WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
    GoCtx = go.run(wasmCtx.instance);

    socket = new WebSocket(`ws://${SOCKET_ENDPOINT}/ws`)
    socket.onopen = function () {
        setInterval(() => this.send("ping"), 2000)
    }
    socket.onmessage = onupdate
}