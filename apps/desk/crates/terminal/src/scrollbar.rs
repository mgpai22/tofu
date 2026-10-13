use gpui::{Bounds, ColorExt, Hsla, PaintQuad, Pixels, fill, point, px, size};

const THUMB_WIDTH: f32 = 4.;
const THUMB_HOT: f32 = 6.;
const THUMB_INSET: f32 = 2.;
const THUMB_MIN: f32 = 24.;
const THUMB_ALPHA: f32 = 0.35;

#[derive(Clone, Copy)]
pub(crate) struct Scrollback {
    pub offset: usize,
    pub history: usize,
    pub rows: usize,
}

pub(crate) fn rail(view: Bounds<Pixels>, gutter: Pixels) -> Bounds<Pixels> {
    Bounds::new(
        point(view.right(), view.top()),
        size(gutter, view.size.height),
    )
}

fn track(view: Bounds<Pixels>) -> Pixels {
    view.size.height - px(2. * THUMB_INSET)
}

fn length(view: Bounds<Pixels>, back: Scrollback) -> Pixels {
    let total = (back.history + back.rows) as f32;
    (track(view) * (back.rows as f32 / total))
        .max(px(THUMB_MIN))
        .min(track(view))
}

pub(crate) fn thumb(
    view: Bounds<Pixels>,
    gutter: Pixels,
    back: Scrollback,
) -> Option<Bounds<Pixels>> {
    if back.history == 0 {
        return None;
    }
    let length = length(view, back);
    let from_top = 1. - back.offset as f32 / back.history as f32;
    Some(Bounds::new(
        point(
            view.right() + gutter - px(THUMB_INSET + THUMB_WIDTH),
            view.top() + px(THUMB_INSET) + (track(view) - length) * from_top,
        ),
        size(px(THUMB_WIDTH), length),
    ))
}

pub(crate) fn offset_at(view: Bounds<Pixels>, back: Scrollback, thumb_top: Pixels) -> usize {
    let travel = track(view) - length(view, back);
    if travel <= px(0.) {
        return back.offset;
    }
    let from_top = ((thumb_top - view.top() - px(THUMB_INSET)) / travel).clamp(0., 1.);
    ((1. - from_top) * back.history as f32).round() as usize
}

pub(crate) fn quad(thumb: Bounds<Pixels>, grabbed: bool, color: Hsla) -> PaintQuad {
    let grow = px(if grabbed { THUMB_HOT - THUMB_WIDTH } else { 0. });
    let bounds = Bounds::new(
        point(thumb.left() - grow, thumb.top()),
        size(thumb.size.width + grow, thumb.size.height),
    );
    fill(bounds, color.opacity(THUMB_ALPHA)).corner_radii(bounds.size.width / 2.)
}
