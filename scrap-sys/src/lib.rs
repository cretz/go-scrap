extern crate libc;

use scrap;

#[no_mangle]
pub unsafe extern "C" fn error_free(err: *mut libc::c_char) {
    std::ffi::CString::from_raw(err);
}

#[repr(C)]
pub struct DisplayListOrErr {
    list: *mut *const scrap::Display,
    len: usize,
    err: *mut libc::c_char,
}

#[no_mangle]
pub unsafe extern "C" fn display_list() -> DisplayListOrErr {
    let mut list = DisplayListOrErr {
        list: std::ptr::null_mut(),
        len: 0,
        err: std::ptr::null_mut(),
    };
    match scrap::Display::all() {
        Ok(displays) => {
            let mut ptrs: Vec<*const scrap::Display> = displays.iter().map(|d| {
                d as *const scrap::Display
            }).collect();
            ptrs.shrink_to_fit();
            list.list = ptrs.as_mut_ptr();
            list.len = ptrs.len();
            std::mem::forget(ptrs);
        }
        Err(err) => {
            list.err = std::ffi::CString::new(err.to_string()).unwrap().into_raw();
        }
    };
    list
}

#[repr(C)]
pub struct DisplayOrErr {
    display: *mut scrap::Display,
    err: *mut libc::c_char,
}

#[no_mangle]
pub unsafe extern "C" fn display_primary() -> DisplayOrErr {
    let mut display = DisplayOrErr { display: std::ptr::null_mut(), err: std::ptr::null_mut() };
    match scrap::Display::primary() {
        Ok(primary) => {
            display.display = Box::into_raw(Box::new(primary))
        }
        Err(err) => {
            display.err = std::ffi::CString::new(err.to_string()).unwrap().into_raw();
        }
    };
    display
}

#[no_mangle]
pub unsafe extern "C" fn get_display(index: libc::c_int) -> DisplayOrErr {
    let mut display = DisplayOrErr { display: std::ptr::null_mut(), err: std::ptr::null_mut() };
    match scrap::Display::all() {
        Ok(displays) => {
            if index as usize < displays.len() {
                display.display = Box::into_raw(Box::new(displays.into_iter().nth(index as usize).unwrap()))
            }
            else {
                display.err = std::ffi::CString::new("No display found in this index").unwrap().into_raw();
            }
        }
        Err(err) => {
            display.err = std::ffi::CString::new(err.to_string()).unwrap().into_raw();
        }
    };
    display
}

#[no_mangle]
pub unsafe extern "C" fn display_free(d: *mut scrap::Display) {
    Box::from_raw(d);
}

#[no_mangle]
pub unsafe extern "C" fn display_width(d: *mut scrap::Display) -> usize {
    (*d).width()
}

#[no_mangle]
pub unsafe extern "C" fn display_height(d: *mut scrap::Display) -> usize {
    (*d).height()
}

#[repr(C)]
pub struct CapturerOrErr {
    capturer: *mut scrap::Capturer,
    err: *mut libc::c_char,
}

#[no_mangle]
pub unsafe extern "C" fn capturer_new(d: *mut scrap::Display) -> CapturerOrErr {
    let display = *Box::from_raw(d);
    let mut ret = CapturerOrErr { capturer: std::ptr::null_mut(), err: std::ptr::null_mut() };
    match scrap::Capturer::new(display) {
        Ok(capturer) => {
            ret.capturer = Box::into_raw(Box::new(capturer))
        }
        Err(err) => {
            ret.err = std::ffi::CString::new(err.to_string()).unwrap().into_raw();
        }
    }
    ret
}

#[no_mangle]
pub unsafe extern "C" fn capturer_free(c: *mut scrap::Capturer) {
    Box::from_raw(c);
}

#[no_mangle]
pub unsafe extern "C" fn capturer_width(c: *mut scrap::Capturer) -> usize {
    (*c).width()
}

#[no_mangle]
pub unsafe extern "C" fn capturer_height(c: *mut scrap::Capturer) -> usize {
    (*c).height()
}

#[repr(C)]
pub struct FrameOrErr {
    data: *const u8,
    len: usize,
    would_block: u8,
    err: *mut libc::c_char,
}

#[no_mangle]
pub unsafe extern "C" fn capturer_frame(c: *mut scrap::Capturer) -> FrameOrErr {
    let mut ret = FrameOrErr {
        data: std::ptr::null_mut(),
        len: 0,
        would_block: 0,
        err: std::ptr::null_mut(),
    };
    let c = &mut *c;
    match c.frame() {
        Ok(frame) => {
            ret.data = frame.as_ptr();
            ret.len = frame.len();
            std::mem::forget(frame);
        }
        Err(ref err) if err.kind() == std::io::ErrorKind::WouldBlock => {
            ret.would_block = 1;
        }
        Err(err) => {
            ret.err = std::ffi::CString::new(err.to_string()).unwrap().into_raw();
        }
    }
    ret
}
