import Jetkvm
import UIKit

/// Invisible responder that brings up the on-screen keyboard and forwards what
/// is typed to the Go app, which draws its own text fields.
final class TextInputProxy: UIView, UIKeyInput {
    var autocorrectionType: UITextAutocorrectionType = .no
    var autocapitalizationType: UITextAutocapitalizationType = .none
    var spellCheckingType: UITextSpellCheckingType = .no
    var smartQuotesType: UITextSmartQuotesType = .no
    var smartDashesType: UITextSmartDashesType = .no
    var smartInsertDeleteType: UITextSmartInsertDeleteType = .no

    override var canBecomeFirstResponder: Bool { true }

    /// Always true, so the keyboard keeps delivering Delete presses; the Go
    /// app owns the actual text.
    var hasText: Bool { true }

    func insertText(_ text: String) {
        var pending = ""
        for character in text {
            if character == "\n" || character == "\r" {
                flush(&pending)
                JetkvmKeyTap(Int(JetkvmKeyEnter))
            } else if character == "\t" {
                flush(&pending)
                JetkvmKeyTap(Int(JetkvmKeyTab))
            } else {
                pending.append(character)
            }
        }
        flush(&pending)
    }

    func deleteBackward() {
        JetkvmKeyTap(Int(JetkvmKeyBackspace))
    }

    private func flush(_ pending: inout String) {
        guard !pending.isEmpty else { return }
        JetkvmInsertText(pending)
        pending = ""
    }
}
