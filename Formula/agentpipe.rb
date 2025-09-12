class Agentpipe < Formula
  desc "Orchestrate conversations between multiple AI CLI agents"
  homepage "https://github.com/kevinelliott/agentpipe"
  version "0.1.0"
  license "MIT"

  # Pre-built binaries for different platforms
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/kevinelliott/agentpipe/releases/download/v0.0.1/agentpipe_darwin_arm64.tar.gz"
      sha256 "62672473179134b0003364f002b6552d78f03e5ab52eaad3ca50cd803bc6002b"
    else
      url "https://github.com/kevinelliott/agentpipe/releases/download/v0.0.1/agentpipe_darwin_amd64.tar.gz"
      sha256 "2bcf605f092ec35ea24a83934b6e15fabdcec61abe81aef1474038b816161aff"
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/kevinelliott/agentpipe/releases/download/v0.0.1/agentpipe_linux_arm64.tar.gz"
      sha256 "746ed129422edc08e864858b05aeb92c9c51660fc3d28f81f3e424c9e6cab626"
    else
      url "https://github.com/kevinelliott/agentpipe/releases/download/v0.0.1/agentpipe_linux_amd64.tar.gz"
      sha256 "2b96b8b5484c009c8d15b7907447a3f2f583c11767b90442069c056bae710c30"
    end
  end

  # Allow building from HEAD
  head "https://github.com/kevinelliott/agentpipe.git", branch: "main"

  # Only need Go for building from HEAD
  depends_on "go" => :build if build.head?

  def install
    if build.head?
      # Build from source
      system "go", "build", *std_go_args(ldflags: "-s -w -X main.Version=#{version}")
    else
      # Install pre-built binary
      # The tar.gz contains the binary directly
      bin.install "agentpipe_#{OS.kernel_name.downcase}_#{Hardware::CPU.arch}"
      
      # Rename to just "agentpipe"
      mv bin/"agentpipe_#{OS.kernel_name.downcase}_#{Hardware::CPU.arch}", bin/"agentpipe"
    end
  end

  def caveats
    <<~EOS
      #{Tty.green}✨ AgentPipe has been installed!#{Tty.reset}
      
      #{Tty.bold}To get started:#{Tty.reset}
        1. Check available AI agents: #{Tty.cyan}agentpipe doctor#{Tty.reset}
        2. Run example: #{Tty.cyan}agentpipe run -a claude:Alice -a gemini:Bob -p "Hello!"#{Tty.reset}
        3. View help: #{Tty.cyan}agentpipe --help#{Tty.reset}
      
      #{Tty.bold}Chat logs are saved to:#{Tty.reset}
        ~/.agentpipe/chats/
      
      #{Tty.bold}Required AI CLI tools:#{Tty.reset}
        - Claude Code CLI: #{Tty.blue}https://github.com/anthropics/claude-code#{Tty.reset}
        - Gemini CLI: #{Tty.blue}https://github.com/google/generative-ai-cli#{Tty.reset}
        - Qwen Code CLI: #{Tty.blue}https://github.com/QwenLM/qwen-code#{Tty.reset}
        - Codex CLI: #{Tty.blue}https://github.com/openai/codex-cli#{Tty.reset}
        - Ollama: #{Tty.blue}https://github.com/ollama/ollama#{Tty.reset}
    EOS
  end

  test do
    # Test doctor command
    output = shell_output("#{bin}/agentpipe doctor 2>&1")
    assert_match "AgentPipe Doctor", output
    
    # Test help
    help_output = shell_output("#{bin}/agentpipe --help 2>&1")
    assert_match "orchestrates conversations", help_output
  end
end
