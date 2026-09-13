import GameController
import Jetkvm
import UIKit

/// Root controller. Keeps the game view above the on-screen keyboard and shows
/// the keyboard while the Go app wants typing and no hardware keyboard is
/// attached (with one attached, Ebitengine already receives key presses).
final class HostViewController: UIViewController {
    private let game = GameViewController()
    private let textInput = TextInputProxy(frame: .zero)
    private var keyboardTimer: Timer?
    private var resigningKeyboard = false

    override var childViewControllerForPointerLock: UIViewController? { game }
    override var childForHomeIndicatorAutoHidden: UIViewController? { game }
    override var childForScreenEdgesDeferringSystemGestures: UIViewController? { game }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .black

        addChild(game)
        // Without the keyboard, extend to the bottom edge instead of stopping
        // above the home indicator.
        view.keyboardLayoutGuide.usesBottomSafeArea = false
        game.view.translatesAutoresizingMaskIntoConstraints = false
        view.addSubview(game.view)
        NSLayoutConstraint.activate([
            game.view.topAnchor.constraint(equalTo: view.topAnchor),
            game.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            game.view.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            game.view.bottomAnchor.constraint(equalTo: view.keyboardLayoutGuide.topAnchor),
        ])
        game.didMove(toParent: self)

        view.addSubview(textInput)

        NotificationCenter.default.addObserver(self, selector: #selector(keyboardWillHide(_:)), name: UIResponder.keyboardWillHideNotification, object: nil)

        let timer = Timer(timeInterval: 0.1, repeats: true) { [weak self] _ in
            self?.syncKeyboard()
        }
        RunLoop.main.add(timer, forMode: .common)
        keyboardTimer = timer
    }

    deinit {
        keyboardTimer?.invalidate()
    }

    func suspendGame() {
        game.suspendGame()
    }

    func resumeGame() {
        game.resumeGame()
    }

    private func syncKeyboard() {
        // GameController must not be touched before Ebitengine is rendering;
        // see GameViewController.installHIDInputIfReady.
        guard game.hidInputReady else { return }
        let wanted = JetkvmTextInputActive() && GCKeyboard.coalesced == nil
        if wanted && !textInput.isFirstResponder {
            textInput.becomeFirstResponder()
        } else if !wanted && textInput.isFirstResponder {
            resigningKeyboard = true
            textInput.resignFirstResponder()
            resigningKeyboard = false
        }
    }

    @objc private func keyboardWillHide(_ notification: Notification) {
        if textInput.isFirstResponder && !resigningKeyboard {
            JetkvmTextInputDismissed()
        }
    }
}
