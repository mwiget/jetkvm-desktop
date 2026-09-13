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
