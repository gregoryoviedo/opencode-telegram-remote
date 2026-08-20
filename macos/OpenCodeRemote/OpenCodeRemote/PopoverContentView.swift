import SwiftUI

struct PopoverContentView: View {
    @ObservedObject var appState: AppState
    let onToggle: () -> Void
    let onSettings: () -> Void
    let onOpenLog: () -> Void
    let onQuit: () -> Void

    private var uptimeText: String? {
        guard case .running = appState.status, let started = appState.startedAt else { return nil }
        let interval = Date().timeIntervalSince(started)
        let formatter = DateComponentsFormatter()
        formatter.allowedUnits = [.hour, .minute, .second]
        formatter.unitsStyle = .abbreviated
        formatter.maximumUnitCount = 2
        return formatter.string(from: interval)
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            header
            Divider()
            statusSection
            if let error = appState.lastError {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            Divider()
            VStack(spacing: 0) {
                actionRow(title: "Configuración…", systemImage: "gear", action: onSettings)
                actionRow(title: "Abrir registro", systemImage: "doc.text", action: onOpenLog)
                actionRow(title: "Salir de OpenCode Remote", systemImage: "power", action: onQuit)
            }
        }
        .padding(16)
        .frame(width: 300)
    }

    private var header: some View {
        HStack {
            Text("OpenCode Remote")
                .font(.headline)
            Spacer()
            Toggle("", isOn: Binding(
                get: { appState.status.isRunning },
                set: { _ in onToggle() }
            ))
            .labelsHidden()
            .toggleStyle(.switch)
            .disabled(!appState.configuration.isValid)
        }
    }

    private var statusSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(appState.status.label)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            if let uptime = uptimeText {
                Text("Tiempo activo \(uptime)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
            if !appState.configuration.isValid {
                if let msg = appState.configuration.validationMessage {
                    Text(msg)
                        .font(.caption)
                        .foregroundStyle(.orange)
                }
            }
        }
    }

    private func actionRow(title: String, systemImage: String, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            HStack {
                Image(systemName: systemImage)
                    .frame(width: 18)
                Text(title)
                Spacer()
            }
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .padding(.vertical, 6)
    }
}