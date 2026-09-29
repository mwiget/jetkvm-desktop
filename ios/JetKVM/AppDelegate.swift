import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    /// UIKit leaves Quit out of the app menu of a "Designed for iPad" app on the
    /// Mac and of the iPadOS menu bar, so add one.
    override func buildMenu(with builder: UIMenuBuilder) {
        super.buildMenu(with: builder)
        guard builder.system == .main else { return }
        let quit = UIKeyCommand(
            title: "Quit JetKVM",
            action: #selector(quit(_:)),
            input: "q",
            modifierFlags: .command
        )
        builder.insertChild(UIMenu(options: .displayInline, children: [quit]), atEndOfMenu: .application)
    }

    @objc private func quit(_ sender: Any?) {
        exit(0)
    }
}
