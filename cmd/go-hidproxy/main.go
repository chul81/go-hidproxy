package main

// Go implementation of Bluetooth to USB HID proxy
// Author: Taneli Leppä <rosmo@rosmo.fi>
// Licensed under Apache License 2.0

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	evdev "github.com/gvalkov/golang-evdev"
	udev "github.com/jochenvg/go-udev"
	"github.com/loov/hrtime"
	"github.com/muka/go-bluetooth/api"
	"github.com/muka/go-bluetooth/bluez/profile/adapter"
	log "github.com/sirupsen/logrus"
	orderedmap "github.com/wk8/go-ordered-map"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type InputDevice struct {
	Device string
	Name   string
}

type InputMessage struct {
	Message   []byte
	Timestamp time.Duration
}

var Scancodes = map[uint16]uint16{
	1: 	41, // KEY_ESC
	2: 	30, // KEY_1
	3: 	31, // KEY_2
	4: 	32, // KEY_3
	5: 	33, // KEY_4
	6: 	34, // KEY_5
	7: 	35, // KEY_6
	8: 	36, // KEY_7
	9: 	37, // KEY_8
	10: 	38, // KEY_9
	11: 	39, // KEY_0
	12: 	45, // KEY_MINUS
	13: 	46, // KEY_EQUAL
	14: 	42, // KEY_BACKSPACE
	15: 	43, // KEY_TAB
	16: 	20, // KEY_Q
	17: 	26, // KEY_W
	18: 	8, // KEY_E
	19: 	21, // KEY_R
	20: 	23, // KEY_T
	21: 	28, // KEY_Y
	22: 	24, // KEY_U
	23: 	12, // KEY_I
	24: 	18, // KEY_O
	25: 	19, // KEY_P
	26: 	47, // KEY_LEFTBRACE
	27: 	48, // KEY_RIGHTBRACE
	28: 	40, // KEY_ENTER
	29: 	224, // KEY_LEFTCTRL
	30: 	4, // KEY_A
	31: 	22, // KEY_S
	32: 	7, // KEY_D
	33: 	9, // KEY_F
	34: 	10, // KEY_G
	35: 	11, // KEY_H
	36: 	13, // KEY_J
	37: 	14, // KEY_K
	38: 	15, // KEY_L
	39: 	51, // KEY_SEMICOLON
	40: 	52, // KEY_APOSTROPHE
	41: 	53, // KEY_GRAVE
	42: 	225, // KEY_LEFTSHIFT
	43: 	49, // KEY_BACKSLASH
	44: 	29, // KEY_Z
	45: 	27, // KEY_X
	46: 	6, // KEY_C
	47: 	25, // KEY_V
	48: 	5, // KEY_B
	49: 	17, // KEY_N
	50: 	16, // KEY_M
	51: 	54, // KEY_COMMA
	52: 	55, // KEY_DOT
	53: 	56, // KEY_SLASH
	54: 	229, // KEY_RIGHTSHIFT
	55: 	85, // KEY_KPASTERISK
	56: 	226, // KEY_LEFTALT
	57: 	44, // KEY_SPACE
	58: 	57, // KEY_CAPSLOCK
	59: 	58, // KEY_F1
	60: 	59, // KEY_F2
	61: 	60, // KEY_F3
	62: 	61, // KEY_F4
	63: 	62, // KEY_F5
	64: 	63, // KEY_F6
	65: 	64, // KEY_F7
	66: 	65, // KEY_F8
	67: 	66, // KEY_F9
	68: 	67, // KEY_F10
	69: 	83, // KEY_NUMLOCK
	70: 	71, // KEY_SCROLLLOCK
	71: 	95, // KEY_KP7
	72: 	96, // KEY_KP8
	73: 	97, // KEY_KP9
	74: 	86, // KEY_KPMINUS
	75: 	92, // KEY_KP4
	76: 	93, // KEY_KP5
	77: 	94, // KEY_KP6
	78: 	87, // KEY_KPPLUS
	79: 	89, // KEY_KP1
	80: 	90, // KEY_KP2
	81: 	91, // KEY_KP3
	82: 	98, // KEY_KP0
	83: 	99, // KEY_KPDOT
	85: 	148, // KEY_ZENKAKUHANKAKU
	86: 	100, // KEY_102ND
	87: 	68, // KEY_F11
	88: 	69, // KEY_F12
	89: 	135, // KEY_RO
	90: 	146, // KEY_KATAKANA
	91: 	147, // KEY_HIRAGANA
	92: 	138, // KEY_HENKAN
	93: 	136, // KEY_KATAKANAHIRAGANA
	94: 	139, // KEY_MUHENKAN
	95: 	140, // KEY_KPJPCOMMA
	96: 	88, // KEY_KPENTER
	97: 	228, // KEY_RIGHTCTRL
	98: 	84, // KEY_KPSLASH
	99: 	70, // KEY_SYSRQ
	100: 	230, // KEY_RIGHTALT
	102: 	74, // KEY_HOME
	103: 	82, // KEY_UP
	104: 	75, // KEY_PAGEUP
	105: 	80, // KEY_LEFT
	106: 	79, // KEY_RIGHT
	107: 	77, // KEY_END
	108: 	81, // KEY_DOWN
	109: 	78, // KEY_PAGEDOWN
	110: 	73, // KEY_INSERT
	111: 	76, // KEY_DELETE
	113: 	127, // KEY_MUTE
	114: 	129, // KEY_VOLUMEDOWN
	115: 	128, // KEY_VOLUMEUP
	116: 	102, // KEY_POWER
	117: 	103, // KEY_KPEQUAL
	119: 	72, // KEY_PAUSE
	121: 	133, // KEY_KPCOMMA
	122: 	144, // KEY_HANGEUL
	123: 	145, // KEY_HANJA
	124: 	137, // KEY_YEN
	125: 	227, // KEY_LEFTMETA
	126: 	231, // KEY_RIGHTMETA
	127: 	101, // KEY_COMPOSE
	128: 	120, // KEY_STOP
	129: 	121, // KEY_AGAIN
	130: 	118, // KEY_PROPS
	131: 	122, // KEY_UNDO
	132: 	119, // KEY_FRONT
	133: 	124, // KEY_COPY
	134: 	116, // KEY_OPEN
	135: 	125, // KEY_PASTE
	136: 	126, // KEY_FIND
	137: 	123, // KEY_CUT
	138: 	117, // KEY_HELP
	140: 	251, // KEY_CALC
	142: 	248, // KEY_SLEEP
	150: 	240, // KEY_WWW
	152: 	249, // KEY_COFFEE
	158: 	241, // KEY_BACK
	159: 	242, // KEY_FORWARD
	161: 	236, // KEY_EJECTCD
	163: 	235, // KEY_NEXTSONG
	164: 	232, // KEY_PLAYPAUSE
	165: 	234, // KEY_PREVIOUSSONG
	166: 	233, // KEY_STOPCD
	173: 	250, // KEY_REFRESH
	176: 	247, // KEY_EDIT
	177: 	245, // KEY_SCROLLUP
	178: 	246, // KEY_SCROLLDOWN
	179: 	182, // KEY_KPLEFTPAREN
	180: 	183, // KEY_KPRIGHTPAREN
	183: 	104, // KEY_F13
	184: 	105, // KEY_F14
	185: 	106, // KEY_F15
	186: 	107, // KEY_F16
	187: 	108, // KEY_F17
	188: 	109, // KEY_F18
	189: 	110, // KEY_F19
	190: 	111, // KEY_F20
	191: 	112, // KEY_F21
	192: 	113, // KEY_F22
	193: 	114, // KEY_F23
	194: 	115, // KEY_F24
}

const (
	RIGHT_META    = 1 << 7
	RIGHT_ALT     = 1 << 6
	RIGHT_SHIFT   = 1 << 5
	RIGHT_CONTROL = 1 << 4
	LEFT_META     = 1 << 3
	LEFT_ALT      = 1 << 2
	LEFT_SHIFT    = 1 << 1
	LEFT_CONTROL  = 1 << 0

	BUTTON_LEFT   = 1 << 0
	BUTTON_RIGHT  = 1 << 1
	BUTTON_MIDDLE = 1 << 2
)

func SetupUSBGadget() {
	const gadget string = "g1" // name of  usb_gadget
	var basepath string = "/sys/kernel/config/usb_gadget/"+gadget
	var paths = []string{
		basepath,
		basepath+"/strings/0x409",
		basepath+"/configs/c.1/strings/0x409",
		basepath+"/functions/hid.usb0",
		basepath+"/functions/hid.usb1",
		basepath+"/os_desc",
	}
	filesStr := orderedmap.New()
	filesStr.Set(basepath+"/idVendor", "0x1d6b") 	//Linux Foundation
	filesStr.Set(basepath+"/idProduct", "0x0104")	//Multifunction Composite Gadget
	filesStr.Set(basepath+"/bcdDevice", "0x0100")
	filesStr.Set(basepath+"/bcdDevice", "0x0100")
	filesStr.Set(basepath+"/bcdUSB", "0x0200")
	filesStr.Set(basepath+"/bDeviceClass", "0xEF")
	filesStr.Set(basepath+"/bDeviceSubClass", "0x02")
	filesStr.Set(basepath+"/bDeviceProtocol", "0x01")
	filesStr.Set(basepath+"/os_desc/use", "1")
	filesStr.Set(basepath+"/os_desc/b_vendor_code", "0x01")
	filesStr.Set(basepath+"/os_desc/qw_sign", "MSFT100")
	filesStr.Set(basepath+"/strings/0x409/serialnumber", "00100")
	filesStr.Set(basepath+"/strings/0x409/manufacturer", "Linux Foundation")
	filesStr.Set(basepath+"/strings/0x409/product", "Multifunction Composite Gadget")
	filesStr.Set(basepath+"/configs/c.1/strings/0x409/configuration", "Config 1: USB Gadget")
	filesStr.Set(basepath+"/configs/c.1/MaxPower", "250")
	filesStr.Set(basepath+"/functions/hid.usb0/protocol", "1")
	filesStr.Set(basepath+"/functions/hid.usb0/subclass", "1")
	filesStr.Set(basepath+"/functions/hid.usb0/report_length", "8")
	filesStr.Set(basepath+"/functions/hid.usb1/protocol", "2")
	filesStr.Set(basepath+"/functions/hid.usb1/subclass", "1")
	filesStr.Set(basepath+"/functions/hid.usb1/report_length", "4")
	var filesBytes = map[string][]byte{
		basepath+"/functions/hid.usb0/report_desc": []byte{0x05, 0x01, 0x09, 0x06, 0xa1, 0x01, 0x05, 0x07, 0x19, 0xe0, 0x29, 0xe7, 0x15, 0x00, 0x25, 0x01, 0x75, 0x01, 0x95, 0x08, 0x81, 0x02, 0x95, 0x01, 0x75, 0x08, 0x81, 0x03, 0x95, 0x05, 0x75, 0x01, 0x05, 0x08, 0x19, 0x01, 0x29, 0x05, 0x91, 0x02, 0x95, 0x01, 0x75, 0x03, 0x91, 0x03, 0x95, 0x06, 0x75, 0x08, 0x15, 0x00, 0x25, 0x65, 0x05, 0x07, 0x19, 0x00, 0x29, 0x65, 0x81, 0x00, 0xc0},
		basepath+"/functions/hid.usb1/report_desc": []byte{0x05, 0x01, 0x09, 0x02, 0xa1, 0x01, 0x09, 0x01, 0xa1, 0x00, 0x05, 0x09, 0x19, 0x01, 0x29, 0x05, 0x15, 0x00, 0x25, 0x01, 0x95, 0x05, 0x75, 0x01, 0x81, 0x02, 0x95, 0x01, 0x75, 0x03, 0x81, 0x01, 0x05, 0x01, 0x09, 0x30, 0x09, 0x31, 0x09, 0x38, 0x15, 0x81, 0x25, 0x7f, 0x75, 0x08, 0x95, 0x03, 0x81, 0x06, 0xc0, 0xc0},
	}
	var symlinks = map[string]string{
		basepath+"/functions/hid.usb0": basepath+"/configs/c.1/hid.usb0",
		basepath+"/functions/hid.usb1": basepath+"/configs/c.1/hid.usb1",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Debugf("Creating directory: %s", path)
			err := os.MkdirAll(path, os.ModeDir)
			if err != nil {
				log.Fatalf("Failed to create directory path: %s", path)
			}
		}
	}

	for pair := filesStr.Oldest(); pair != nil; pair = pair.Next() {
		content, err := ioutil.ReadFile(pair.Key.(string))
		if err == nil {
			if bytes.Compare(content[0:len(content)-1], []byte(pair.Value.(string))) == 0 {
				continue
			}
		}

		log.Debugf("Writing file: %s", pair.Key.(string))
		err = ioutil.WriteFile(pair.Key.(string), []byte(pair.Value.(string)), os.FileMode(0644))
		if err != nil {
			log.Warnf("Failed to write file: %s (maybe already set up)", pair.Key.(string))
		}
	}

	for file, contents := range filesBytes {
		content, err := ioutil.ReadFile(file)
		if err == nil {
			if bytes.Compare(content, contents) == 0 {
				continue
			}
		}
		log.Debugf("Writing file: %s", file)
		err = ioutil.WriteFile(file, contents, os.FileMode(0644))
		if err != nil {
			log.Warnf("Failed to create file: %s (maybe already set up)", file)
		}
	}

	for source, target := range symlinks {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			log.Debugf("Creating symlink from %s to: %s", source, target)
			err := os.Symlink(source, target)
			if err != nil {
				log.Fatalf("Failed to create symlink %s -> %s", source, target)
			}
		}
	}

	time.Sleep(1000 * time.Millisecond)

	matches, err := filepath.Glob("/sys/class/udc/*")
	if err != nil {
		log.Fatalf("Failed to list files in /sys/class/udc: %s", err.Error())
	}
	var udcFile string = basepath+"/UDC"
	var udc string = ""
	for _, match := range matches {
		udc = udc + filepath.Base(match) + " "
	}
	content, err := ioutil.ReadFile(udcFile)
	if err == nil {
		if bytes.Compare(content[0:len(content)-1], []byte(strings.TrimSpace(udc))) != 0 {
			err = ioutil.WriteFile(udcFile, []byte(strings.TrimSpace(udc)), os.FileMode(0644))
			if err != nil {
				log.Warnf("Failed to create file %s: %s: (%s)", udcFile, udc, err.Error())
			}
		}
	}
	// Give it a second to settle
	time.Sleep(1000 * time.Millisecond)
}

func HandleKeyboard(output chan<- error, input chan<- InputMessage, close <-chan bool, rate uint, delay uint, dev evdev.InputDevice) error {
	keysDown := make([]uint16, 0)
	err := dev.Grab()
	if err != nil {
		log.Fatal(err)
		output <- err
		return err
	}
	defer dev.Release()

	log.Infof("Grabbed keyboard-like device: %s (%s)", dev.Name, dev.Fn)
	syscall.SetNonblock(int(dev.File.Fd()), true)

	log.Infof("Setting repeat rate to %d, delay %d for %s (%s)", rate, delay, dev.Name, dev.Fn)
	dev.SetRepeatRate(rate, delay)

	loop := 0
	for {
		err = dev.File.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		if err != nil {
			log.Fatal(err)
			output <- err
			return err
		}

		event, err := dev.ReadOne()
		if err != nil && strings.Contains(err.Error(), "i/o timeout") {
			continue
		}
		if err != nil {
			log.Fatal(err)
			output <- err
			return err
		}
		log.Debugf("Keyboard input event: type=%d, code=%d, value=%d", event.Type, event.Code, event.Value)
		if event.Type == evdev.EV_KEY {
			keyEvent := evdev.NewKeyEvent(event)
			log.Debugf("Key event: scancode=%d, keycode=%d, state=%d", keyEvent.Scancode, keyEvent.Keycode, keyEvent.State)
			if keyCode, ok := Scancodes[keyEvent.Scancode]; ok {
				if keyEvent.State == 1 { // Key down
					keyIsDown := false
					for _, k := range keysDown {
						if k == keyCode {
							keyIsDown = true
						}
					}
					if !keyIsDown {
						keysDown = append(keysDown, keyCode)
					}
				}
				if keyEvent.State == 0 { // Key up
					newKeysDown := make([]uint16, 0)
					for _, k := range keysDown {
						if k != keyCode {
							newKeysDown = append(newKeysDown, k)
						}
					}
					keysDown = newKeysDown
				}

				var modifiers uint8 = 0
				keysToSend := make([]uint8, 0)
				for _, k := range keysDown {
					switch {
					case k == 224: // Left-Ctrl
						modifiers |= LEFT_CONTROL
					case k == 227: // Left-Cmd
						modifiers |= LEFT_META
					case k == 225: // Left-Shift
						modifiers |= LEFT_SHIFT
					case k == 226: // Left-Alt
						modifiers |= LEFT_ALT
					case k == 228: // Right-Ctrl
						modifiers |= RIGHT_CONTROL
					case k == 231: // Right-Cmd
						modifiers |= RIGHT_META
					case k == 229: // Right-Shift
						modifiers |= RIGHT_SHIFT
					case k == 230: // Right-Alt
						modifiers |= RIGHT_ALT
					default:
						keysToSend = append(keysToSend, uint8(k))
					}
				}
				keysToSend = append([]uint8{modifiers, 0}, keysToSend...)
				if len(keysToSend) < 8 {
					for i := len(keysToSend); i < 8; i++ {
						keysToSend = append(keysToSend, uint8(0))
					}
				}
				input <- InputMessage{
					Timestamp: hrtime.Now(),
					Message: keysToSend,
				}

				log.Debugf("Key status (scancode %d, keycode %d): %v\n", keyEvent.Scancode, keyCode, keysToSend)
			} else {
				log.Warnf("Unknown scancode: %d\n", keyEvent.Scancode)
			}
		}
		loop += 1
		if loop > 3 {
			select {
			case _ = <-close:
				log.Infof("Stopping processing keyboard input from: %s (%s)", dev.Name, dev.Fn)
				output <- nil
				return nil
			default:
			}
			loop = 0
		}
	}

	output <- nil
	return nil
}

func HandleMouse(output chan<- error, input chan<- InputMessage, close <-chan bool, dev evdev.InputDevice) error {
	err := dev.Grab()
	if err != nil {
		log.Fatal(err)
		output <- err
		return err
	}
	defer dev.Release()

	log.Infof("Grabbed mouse-like device: %s (%s)", dev.Name, dev.Fn)
	syscall.SetNonblock(int(dev.File.Fd()), true)

	loop := 0
	var buttons uint8 = 0x0
	for {
		err = dev.File.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		if err != nil {
			log.Fatal(err)
			output <- err
			return err
		}

		event, err := dev.ReadOne()
		if err != nil && strings.Contains(err.Error(), "i/o timeout") {
			continue
		}
		if err != nil {
			log.Fatal(err)
			output <- err
			return err
		}
		log.Debugf("Mouse input event: type=%d, code=%d, value=%d", event.Type, event.Code, event.Value)
		var buttonOp bool = false
		if event.Type == evdev.EV_KEY {
			if event.Code == 272 {
				if event.Value > 0 {
					buttons |= BUTTON_LEFT
				} else {
					buttons &= ^uint8(BUTTON_LEFT)
				}
				buttonOp = true
			}
			if event.Code == 273 {
				if event.Value > 0 {
					buttons |= BUTTON_RIGHT
				} else {
					buttons &= ^uint8(BUTTON_RIGHT)
				}
				buttonOp = true
			}
			if event.Code == 274 {
				if event.Value > 0 {
					buttons |= BUTTON_MIDDLE
				} else {
					buttons &= ^uint8(BUTTON_MIDDLE)
				}
				buttonOp = true
			}
		}
		if event.Type == evdev.EV_REL || buttonOp {
			mouseToSend := make([]uint8, 0)
			mouseToSend = append(mouseToSend, buttons)
			if event.Type == evdev.EV_REL {
				if event.Code == 0 {
					mouseToSend = append(mouseToSend, uint8(event.Value))
					mouseToSend = append(mouseToSend, 0x00)
					mouseToSend = append(mouseToSend, 0x00)
				}
				if event.Code == 1 {
					mouseToSend = append(mouseToSend, 0x00)
					mouseToSend = append(mouseToSend, uint8(event.Value))
					mouseToSend = append(mouseToSend, 0x00)
				}
				if event.Code == 11 {
					mouseToSend = append(mouseToSend, 0x00)
					mouseToSend = append(mouseToSend, 0x00)
					mouseToSend = append(mouseToSend, uint8(event.Value))
				}
			} else {
				mouseToSend = append(mouseToSend, 0x00)
				mouseToSend = append(mouseToSend, 0x00)
				mouseToSend = append(mouseToSend, 0x00)
			}
			input <- InputMessage{
					Timestamp: hrtime.Now(),
					Message: mouseToSend,
				}
		}
		loop += 1
		if loop > 3 {
			select {
			case _ = <-close:
				log.Infof("Stopping processing mouse input from: %s (%s)", dev.Name, dev.Fn)
				output <- nil
				return nil
			default:
			}
			loop = 0
		}
	}

	output <- nil
	return nil

}

func SendKeyboardReports(input <-chan InputMessage) error {
	log.Info("Opening keyboard /dev/hidg0 for writing...")
	file, err := os.OpenFile("/dev/hidg0", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Warn("Error opening /dev/hidg0, are you running as root?")
		log.Fatal(err)
		return err
	}
	defer file.Close()

	var avg, min, max, loop int64 = 0, 0, 0, 0
	for {
		msg := <-input
		bytesWritten, err := file.Write(msg.Message)
		if err != nil {
			log.Fatal(err)
			return err
		} 
		latency := hrtime.Since(msg.Timestamp).Nanoseconds()
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
		avg = (avg + latency) / 2
		loop += 1
		if loop > 50 {
			log.Debugf("Latency: now=%d, avg=%d, min=%d, max=%d μs", latency/1000, avg/1000, min/1000, max/1000)
			loop = 0
		}

		log.Debugf("Wrote %d bytes to /dev/hidg0 (%v)", bytesWritten, msg)
	}
	return nil
}

func SendMouseReports(input <-chan InputMessage) error {
	log.Info("Opening keyboard /dev/hidg1 for writing...")
	file, err := os.OpenFile("/dev/hidg1", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Warn("Error opening /dev/hidg1, are you running as root?")
		log.Fatal(err)
		return err
	}
	defer file.Close()

	var avg, min, max, loop int64 = 0, 0, 0, 0
	for {
		msg := <-input
		bytesWritten, err := file.Write(msg.Message)
		if err != nil {
			log.Fatal(err)
			return err
		}
		log.Debugf("Wrote %d bytes to /dev/hidg1 (%v)", bytesWritten, msg)
		latency := hrtime.Since(msg.Timestamp).Nanoseconds()
		if latency < min {
			min = latency
		}
		if latency > max {
			max = latency
		}
		avg = (avg + latency) / 2
		loop += 1
		if loop > 100 {
			log.Debugf("Latency: now=%d, avg=%d, min=%d, max=%d μs", latency/1000, avg/1000, min/1000, max/1000)
			loop = 0
		}
	}
	return nil
}

func GetDisconnectedDevices(adapterId string) ([]string, error) {
	log.Debugf("Getting adapter: %s", adapterId)
	a, err := adapter.GetAdapter(adapterId)
	if err != nil {
		return nil, err
	}

	log.Debugf("Getting devices from adapter: %s", adapterId)
	devices, err := a.GetDevices()
	if err != nil {
		return nil, err
	}

	disconnected := make([]string, 0)
	connected := make([]string, 0)
	for _, dev := range devices {
		address, err := dev.GetAddress()
		if err != nil {
			continue
		}
		name, err := dev.GetName()
		if err != nil {
			name = "?"
		}

		log.Infof("Checking if device %s (%s) is connected...", name, address)
		deviceConnected, err := dev.GetConnected()
		if err == nil {
			if !deviceConnected {
				log.Infof("Device %s is disconnected.", name)
				disconnected = append(disconnected, name)
			} else {
				log.Infof("Device %s is still connected.", name)
				connected = append(connected, name)
			}
		}
	}
	results := make([]string, 0)
	for _, dname := range disconnected {
		ok := true
		for _, cname := range connected {
			if cname == dname {
				ok = false
				break
			}
		}
		if ok {
			inResults := false
			for _, rname := range results {
				if rname == dname {
					inResults = true
				}
			}
			if !inResults {
				results = append(results, dname)
			}
		}
	}
	return results, nil
}

func main() {
	var wg sync.WaitGroup
	logLevelPtr := flag.String("loglevel", "warn", "log level (panic, fatal, error, warn, info, debug, trace)")
	setupHid := flag.Bool("setuphid", true, "setup HID files on startup")
	setupMouse := flag.Bool("mouse", true, "setup mouse(s)")
	setupKeyboard := flag.Bool("keyboard", true, "setup keyboard(s)")
	monitorUdev := flag.Bool("monitor-udev", true, "monitor udev & BlueZ events for disconnects")
	adapterId := flag.String("bluez-adapter", "hci0", "BlueZ adapter (default hci0)")
	kbdRepeat := flag.Int("kbdrepeat", 62, "set keyboard repeat rate (default 62)")
	kbdDelay := flag.Int("kbddelay", 300, "set keyboard repeat delay in ms (default 300)")
	flag.Parse()

	logLevel, err := log.ParseLevel(*logLevelPtr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Set log level: %v\n", logLevel)
	log.SetLevel(logLevel)

	if *setupHid {
		log.Info("Setting up HID files...")
		SetupUSBGadget()
	}

	keyboardInput := make(chan InputMessage, 10)
	mouseInput := make(chan InputMessage, 100)
	output := make(map[InputDevice]chan error, 0)
	close := make(map[InputDevice]chan bool, 0)

	var udevCh <-chan *udev.Device
	var cancel context.CancelFunc
	var ctx context.Context

	defer api.Exit()
	u := udev.Udev{}
	if *monitorUdev {
		log.Info("Starting udev monitoring for Bluetooth devices")
		m := u.NewMonitorFromNetlink("udev")
		m.FilterAddMatchSubsystem("bluetooth")

		ctx, cancel = context.WithCancel(context.Background())
		udevCh, _ = m.DeviceChan(ctx)
	}

	go SendKeyboardReports(keyboardInput)
	go SendMouseReports(mouseInput)
	wg.Add(1)
	for {
		select {
		case d := <-udevCh:
			if d.Action() == "add" || d.Action() == "remove" {
				disconnected, err := GetDisconnectedDevices(*adapterId)
				if err != nil {
					log.Errorf("Error checking disconnected devices: %s", err.Error())
				} else {
					for _, device := range disconnected {
						for devId, _ := range output {
							if strings.HasPrefix(devId.Name, device) {
								log.Infof("Disconnected device, stopping listening to: %s (%s)", devId.Name, devId.Device)
								select {
								case close[devId] <- true:
									log.Infof("Sent stop signal to: %s (%s)", devId.Name, devId.Device)
								default:
								}

							}
						}
					}
				}
			}
		default:
		}

		//log.Debugf("Polling for new devices in /dev/input")
		devices, _ := evdev.ListInputDevices()
		for _, dev := range devices {
			isMouse := false
			isKeyboard := false
			for k := range dev.Capabilities {
				if k.Name == "EV_REL" {
					isMouse = true
				}
				if k.Name == "EV_KEY" {
					isKeyboard = true
				}
			}
			log.Debugf("Device %s (%s), capabilities: %v (mouse=%t, kbd=%t)", dev.Name, dev.Fn, dev.Capabilities, isMouse, isKeyboard)
			if isKeyboard || isMouse {
				devId := InputDevice{
					Device: dev.Fn,
					Name:   dev.Name,
				}
				if _, ok := output[devId]; !ok {
					output[devId] = make(chan error, 10)
					close[devId] = make(chan bool, 10)
					if isKeyboard && !isMouse && *setupKeyboard {
						go HandleKeyboard(output[devId], keyboardInput, close[devId], uint(*kbdRepeat), uint(*kbdDelay), *dev)
						wg.Add(1)
					}
					log.Debugf("isKeyboard: %t, isMouse: %t, setupMouse: %t", !isKeyboard, isMouse, *setupMouse)
					if isMouse && *setupMouse {
						go HandleMouse(output[devId], mouseInput, close[devId], *dev)
						wg.Add(1)
					}
				}
			}
		}
		time.Sleep(1000 * time.Millisecond)
		for id, eventOutput := range output {
			select {
			case msg := <-eventOutput:
				if msg == nil {
					log.Warnf("Event handler quit: %s", id.Device)
				} else {
					log.Errorf("Received error from %s: %s", id.Device, msg.Error())
				}
				delete(output, id)
				wg.Done()
			default:
			}
		}
	}
	cancel()
}
