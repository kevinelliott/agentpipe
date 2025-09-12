class Agentpipe < Formula
  desc "Orchestrate conversations between multiple AI CLI agents"
  homepage "https://github.com/kevinelliott/agentpipe"
  url "https://github.com/kevinelliott/agentpipe/archive/v0.1.0.tar.gz"
  sha256 ""  # Will be updated when release is created
  license "MIT"
  head "https://github.com/kevinelliott/agentpipe.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w")
  end

  test do
    # Test that the doctor command runs successfully
    output = shell_output("#{bin}/agentpipe doctor 2>&1")
    assert_match "AgentPipe Doctor", output
    
    # Test that the help command works
    help_output = shell_output("#{bin}/agentpipe --help 2>&1")
    assert_match "AgentPipe orchestrates conversations", help_output
  end
end