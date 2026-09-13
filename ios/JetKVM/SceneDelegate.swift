import Jetkvm
import UIKit

final class SceneDelegate: UIResponder, UIWindowSceneDelegate {
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

    func sceneDidBecomeActive(_ scene: UIScene) {
        hostController.resumeGame()
        JetkvmAppDidBecomeActive()
    }

    func sceneWillResignActive(_ scene: UIScene) {
        hostController.suspendGame()
    }

    func sceneDidEnterBackground(_ scene: UIScene) {
        JetkvmAppDidEnterBackground()
    }
}
