import GameController
import Jetkvm
import UIKit

/// Hosts the Ebitengine view and forwards pointer input and appearance to the
/// Go app. Ebitengine only handles touches and keys on iOS; mouse position,
/// buttons, scrolling and pointer lock come from here (see pkg/app/hostinput.go).
final class GameViewController: JetkvmEbitenViewController {
    /// Trackpad and wheel scrolling arrives in points; the app expects wheel units.
    private static let scrollPointsPerWheelUnit = 10.0

    private var activeTouch: UITouch?
    private var lastScrollTranslation = CGPoint.zero
    private var pointerLocked = false
    private var pointerHidden = false
    private var pointerInteraction: UIPointerInteraction?
    private var hidInputInstalled = false
    private var stateTimer: Timer?

    /// Whether scroll and GCMouse input are set up, i.e. Ebitengine is rendering
    /// and GameController may be used.
    var hidInputReady: Bool { hidInputInstalled }

    override var prefersPointerLocked: Bool { pointerLocked }
    override var prefersHomeIndicatorAutoHidden: Bool { true }
    override var preferredScreenEdgesDeferringSystemGestures: UIRectEdge { .all }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.isMultipleTouchEnabled = true

        let hover = UIHoverGestureRecognizer(target: self, action: #selector(handleHover(_:)))
        view.addGestureRecognizer(hover)

        let interaction = UIPointerInteraction(delegate: self)
        view.addInteraction(interaction)
        pointerInteraction = interaction

        registerForTraitChanges([UITraitUserInterfaceStyle.self]) { (controller: GameViewController, _: UITraitCollection) in
            controller.reportAppearance()
        }
        reportAppearance()

        let timer = Timer(timeInterval: 0.1, repeats: true) { [weak self] _ in
            self?.syncPointerState()
        }
        RunLoop.main.add(timer, forMode: .common)
        stateTimer = timer
    }

    deinit {
        stateTimer?.invalidate()
    }

    /// Installs scroll and GCMouse input once Ebitengine has started rendering.
    /// Setting these up in viewDidLoad, before the game view exists, reliably
    /// kept Ebitengine's render thread from starting (black screen) on iOS 27.
    private func installHIDInputIfReady() {
        guard !hidInputInstalled, !view.subviews.isEmpty else { return }
        hidInputInstalled = true

        let scroll = UIPanGestureRecognizer(target: self, action: #selector(handleScroll(_:)))
        scroll.allowedScrollTypesMask = .all
        scroll.allowedTouchTypes = []
        view.addGestureRecognizer(scroll)

        NotificationCenter.default.addObserver(self, selector: #selector(mouseDidConnect(_:)), name: .GCMouseDidConnect, object: nil)
        GCMouse.mice().forEach(attach)
    }

    // MARK: Touches (fingers, and trackpad or mouse clicks)

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent?) {
        super.touchesBegan(touches, with: event)
        guard activeTouch == nil, let touch = touches.first, !ignoresTouch(touch) else { return }
        activeTouch = touch
        reportPosition(of: touch)
        JetkvmPointerButtons(buttons(for: touch, event: event))
    }

    override func touchesMoved(_ touches: Set<UITouch>, with event: UIEvent?) {
        super.touchesMoved(touches, with: event)
        guard let touch = activeTouch, touches.contains(touch) else { return }
        reportPosition(of: touch)
        if touch.type == .indirectPointer {
            JetkvmPointerButtons(buttons(for: touch, event: event))
        }
    }

    override func touchesEnded(_ touches: Set<UITouch>, with event: UIEvent?) {
        super.touchesEnded(touches, with: event)
        endActiveTouch(in: touches)
    }

    override func touchesCancelled(_ touches: Set<UITouch>, with event: UIEvent?) {
        super.touchesCancelled(touches, with: event)
        endActiveTouch(in: touches)
    }

    private func endActiveTouch(in touches: Set<UITouch>) {
        guard let touch = activeTouch, touches.contains(touch) else { return }
        activeTouch = nil
        reportPosition(of: touch)
        JetkvmPointerButtons(0)
    }

    /// While the pointer is locked, mouse buttons and motion come from GCMouse.
    private func ignoresTouch(_ touch: UITouch) -> Bool {
        pointerLocked && touch.type == .indirectPointer
    }

    private func reportPosition(of touch: UITouch) {
        guard !pointerLocked else { return }
        let point = touch.location(in: view)
        JetkvmPointerMoved(Double(point.x), Double(point.y))
    }

    private func buttons(for touch: UITouch, event: UIEvent?) -> Int {
        guard touch.type == .indirectPointer, let mask = event?.buttonMask else { return 1 }
        var buttons = 0
        if mask.contains(.primary) { buttons |= 1 }
        if mask.contains(.secondary) { buttons |= 2 }
        if mask.contains(.button(3)) { buttons |= 4 }
        if mask.contains(.button(4)) { buttons |= 8 }
        if mask.contains(.button(5)) { buttons |= 16 }
        return buttons == 0 ? 1 : buttons
    }

    // MARK: Hover and scrolling

    @objc private func handleHover(_ recognizer: UIHoverGestureRecognizer) {
        guard !pointerLocked, recognizer.state == .began || recognizer.state == .changed else { return }
        let point = recognizer.location(in: view)
        JetkvmPointerMoved(Double(point.x), Double(point.y))
    }

    @objc private func handleScroll(_ recognizer: UIPanGestureRecognizer) {
        switch recognizer.state {
        case .began:
            lastScrollTranslation = .zero
        case .changed:
            let translation = recognizer.translation(in: view)
            let dx = Double(translation.x - lastScrollTranslation.x) / Self.scrollPointsPerWheelUnit
            let dy = Double(translation.y - lastScrollTranslation.y) / Self.scrollPointsPerWheelUnit
            lastScrollTranslation = translation
            JetkvmPointerScrolled(dx, dy)
        default:
            break
        }
    }

    // MARK: Pointer lock (relative mouse mode)

    @objc private func mouseDidConnect(_ notification: Notification) {
        if let mouse = notification.object as? GCMouse {
            attach(mouse)
        }
    }

    private func attach(_ mouse: GCMouse) {
        guard let input = mouse.mouseInput else { return }
        input.mouseMovedHandler = { [weak self] _, dx, dy in
            guard self?.pointerLocked == true else { return }
            JetkvmPointerMovedBy(Double(dx), Double(-dy))
        }
        var buttonInputs: [GCControllerButtonInput?] = [input.leftButton, input.rightButton, input.middleButton]
        buttonInputs += input.auxiliaryButtons ?? []
        for button in buttonInputs {
            button?.pressedChangedHandler = { [weak self] _, _, _ in
                self?.reportLockedButtons(input)
            }
        }
    }

    private func reportLockedButtons(_ input: GCMouseInput) {
        guard pointerLocked else { return }
        var buttons = 0
        if input.leftButton.isPressed { buttons |= 1 }
        if input.rightButton?.isPressed == true { buttons |= 2 }
        if input.middleButton?.isPressed == true { buttons |= 4 }
        let auxiliary = input.auxiliaryButtons ?? []
        if auxiliary.count > 0, auxiliary[0].isPressed { buttons |= 8 }
        if auxiliary.count > 1, auxiliary[1].isPressed { buttons |= 16 }
        JetkvmPointerButtons(buttons)
    }

    private func syncPointerState() {
        installHIDInputIfReady()

        let wantsLock = JetkvmPointerLockRequested()
        if wantsLock != pointerLocked {
            pointerLocked = wantsLock
            if !wantsLock {
                JetkvmPointerButtons(0)
            }
            setNeedsUpdateOfPrefersPointerLocked()
        }
        let hidden = JetkvmPointerHidden()
        if hidden != pointerHidden {
            pointerHidden = hidden
            pointerInteraction?.invalidate()
        }
    }

    // MARK: Appearance

    private func reportAppearance() {
        JetkvmSetDarkMode(traitCollection.userInterfaceStyle == .dark)
    }
}

extension GameViewController: UIPointerInteractionDelegate {
    func pointerInteraction(_ interaction: UIPointerInteraction, styleFor region: UIPointerRegion) -> UIPointerStyle? {
        pointerHidden ? .hidden() : nil
    }
}
