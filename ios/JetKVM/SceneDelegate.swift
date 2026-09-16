import Jetkvm
import UIKit

final class SceneDelegate: UIResponder, UIWindowSceneDelegate {
    /// macOS moves a "Designed for iPad" app's scene all the way to background
    /// as soon as another app comes to the front, even while the window stays
    /// visible. Unlike iPadOS it keeps the process and its WebRTC session
    /// running throughout, so coming back must not throw that session away.
    private static let reconnectsAfterBackground = !ProcessInfo.processInfo.isiOSAppOnMac

    var window: UIWindow?
    private let hostController = HostViewController()

    func scene(_ scene: UIScene, willConnectTo session: UISceneSession, options connectionOptions: UIScene.ConnectionOptions) {
        guard let windowScene = scene as? UIWindowScene else { return }
        let window = UIWindow(windowScene: windowScene)
        window.rootViewController = hostController
        window.makeKeyAndVisible()
        self.window = window
        openURLs(connectionOptions.urlContexts)
    }

    func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
        openURLs(URLContexts)
    }

    /// jetkvm://host[:port] connects to that device.
    private func openURLs(_ contexts: Set<UIOpenURLContext>) {
        for context in contexts {
            JetkvmOpenURL(context.url.absoluteString)
        }
    }

    // The game is suspended out of sight rather than when the scene resigns
    // active: on iPadOS an inactive scene can still be on screen, beside another
    // app in Split View or Stage Manager, and the KVM picture should keep up.

    func sceneWillEnterForeground(_ scene: UIScene) {
        hostController.resumeGame()
        JetkvmAppWillEnterForeground()
    }

    func sceneDidBecomeActive(_ scene: UIScene) {
        JetkvmAppDidBecomeActive()
    }

    func sceneWillResignActive(_ scene: UIScene) {
        JetkvmAppWillResignActive()
    }

    func sceneDidEnterBackground(_ scene: UIScene) {
        hostController.suspendGame()
        JetkvmAppDidEnterBackground(Self.reconnectsAfterBackground)
    }
}
