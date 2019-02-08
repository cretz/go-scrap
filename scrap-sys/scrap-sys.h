
void error_free(char* err);

struct Display;

typedef struct {
    // These have to be freed separately
    struct Display** list;
    size_t len;
    char* err;
} DisplayListOrErr;

DisplayListOrErr display_list();

typedef struct {
    struct Display* display;
    char* err;
} DisplayOrErr;

DisplayOrErr display_primary();

void display_free(struct Display* display);

size_t display_width(struct Display* display);

size_t display_height(struct Display* display);

struct Capturer;

typedef struct {
    struct Capturer* capturer;
    char* err;
} CapturerOrErr;

CapturerOrErr capturer_new(struct Display* display);

void capturer_free(struct Capturer* capturer);

size_t capturer_width(struct Capturer* capturer);

size_t capturer_height(struct Capturer* capturer);

typedef struct {
    unsigned char* data;
    size_t len;
    char would_block;
    char* err;
} FrameOrErr;

FrameOrErr capturer_frame(struct Capturer* capturer);

void frame_free(unsigned char* data, size_t len);