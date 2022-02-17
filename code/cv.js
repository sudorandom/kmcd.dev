import PropTypes from 'prop-types';
import React, { Fragment } from 'react';


/**
 * The partial component
 *
 * @disable-docs
 */
const CV = ({ _body, _relativeURL, _ID, name, description, job, citizenship, residence, email}) => (
	<Fragment>
		<div className="section about">
			<div className="content">
				<div className="title">
					<div className="title_inner">About Me</div>
				</div>
				<div className="image">
					<img src={_relativeURL( `/assets/images/me.png`, _ID )} alt="" />
				</div>
				<div className="desc">
					<p>
						{ description }
					</p>
					<div className="info-list">
						<ul>
							<li><strong>Name:</strong> { name }</li>
							<li><strong>Job:</strong> { job }</li>
							<li><strong>Citizenship:</strong> { citizenship }</li>
							<li><strong>Residence: </strong> { residence }</li>
							<li><strong>E-mail:</strong> { email }</li>
						</ul>
					</div>
				</div>
				<div className="clear"></div>
			</div>
		</div>
		
		<div className="section resume">
			<div className="content">
				<div className="title">
					<div className="title_inner">Experience & Education</div>
				</div>
				<div className="resume-items">
					<div className="resume-item active">
						<div className="date">2021 - present</div>
						<div className="name">Software Engineer - Vital Beats</div>
					</div>
					<div className="resume-item active">
						<div className="date">2016 - 2021</div>
						<div className="name">Software Engineer - Apple, Inc.</div>
						<ul className="resume-sub-item">
							<li>Worked with Apple network engineers to create highly scalable and redundant tooling and services to track, model, monitor, and configure critical network devices. This tooling improved visibility into the operation of Apple's global backbone network which serves hundreds of millions of users.</li>
							<li>Advocated and educated other teams about open standards, like OpenConfig, within Apple and to several of Apple's vendors.</li>
							<li>Programmed primarily in Clojure and Go. Technologies used: TL1, NETCONF, gRPC, gNMI, YANG, OpenConfig, SNMP, LLDP, BGP, om.next, Datomic, Zookeeper, Cassandra, Kafka. Interoperated with Juniper/Arista/Nokia/Cisco/etc. switches and routers.</li>
						</ul>
					</div>
					<div className="resume-item">
						<div className="date">2011 - 2016</div>
						<div className="name">Senior Software Engineer - SoftLayer, an IBM Company</div>
						<ul className="resume-sub-item">
							<li>Maintained several backend systems that collect, store and process large amounts of time series data.}</li>
							<li>Created and maintained SoftLayer’s open source projects including the python language bindings for SoftLayer’s API (softlayer-python) </li>and an SFTP/FTP frontend for SoftLayer’s object storage product (swftp). I have also contributed upstream patches to open source projects like OpenStack.}
							<li>Built a network poller in Go which supported SNMP, TCP, ICMP, HTTP and DNS which was able to handle hundreds of thousands of targets </li>with a single instance.}
							<li>Produced holistic evaluations of SoftLayer’s virtual server, object storage and internal metric system which resulted in a large </li>amount of actionable data for operations and future product design. Due to the success of the project, this approach towards data collection and visualization was emulated in other areas of the company.}
							<li>Programmed primarily in Python, Java, Go, PHP. Technologies used: HBase, Cassandra, InfluxDB, Oracle, RabbitMQ, Kafka, Ansible, Chef, </li>Graphite, Grafana, Splunk, Xenserver, SNMP and OpenStack Ceilometer.}
						</ul>
					</div>
					<div className="resume-item">
						<div className="date">2010 - 2011</div>
						<div className="name">Senior Application Developer - Distribion Inc.</div>
						<ul className="resume-sub-item">
							<li>Positioned in a support role: Debugging time-sensitive problems, solving customer issues, database management, release management. This position required usage of PHP, JavaScript, PostgreSQL, mySQL, git, subversion and an assortment of domain-specific tooling</li>
						</ul>
					</div>
					<div className="resume-item">
						<div className="date">2006 - 2011</div>
						<div className="name">University of Texas at Dallas</div>
						<div className="name">Dallas, Texas, USA</div>
						<p>
							I attained a Bachelor of Sciences in Computer Science. Notable classes were 'Computer Graphics', 'Game Design', 'Object Oriented Programming'
						</p>
					</div>
				</div>
			</div>
		</div>

		<div className="section skills">
			<div className="content">
				<div className="title">
					<div className="title_inner">Coding Skills</div>
				</div>
				<div className="skills circles">
					<ul>
						<li> 
							<div className="name">Go</div>
							<div className="progress p90">
								<div className="percentage" style={{ width: "90%" }}>
									<span className="percent">90%</span>
								</div>
								<span>90%</span>
							</div>
						</li>
						<li> 
							<div className="name">Elixir</div>
							<div className="progress p75">
								<div className="percentage" style={{ width: "75%" }}>
									<span className="percent">75%</span>
								</div>
								<span>75%</span>
							</div>
						</li>
						<li> 
							<div className="name">Clojure</div>
							<div className="progress p80">
								<div className="percentage" style={{ width: "80%" }}>
									<span className="percent">80%</span>
								</div>
								<span>80%</span>
							</div>
						</li>
						<li> 
							<div className="name">Python</div>
							<div className="progress p90">
								<div className="percentage" style={{ width: "90%" }}>
									<span className="percent">90%</span>
								</div>
								<span>90%</span>
							</div>
						</li>
						<li> 
							<div className="name">Java</div>
							<div className="progress p70">
								<div className="percentage" style={{ width: "70%" }}>
									<span className="percent">70%</span>
								</div>
								<span>70%</span>
							</div>
						</li>
						<li> 
							<div className="name">PHP</div>
							<div className="progress p60">
								<div className="percentage" style={{ width: "60%" }}>
									<span className="percent">60%</span>
								</div>
								<span>60%</span>
							</div>
						</li>
						<li> 
							<div className="name">JavaScript</div>
							<div className="progress p50">
								<div className="percentage" style={{ width: "50%" }}>
									<span className="percent">50%</span>
								</div>
								<span>50%</span>
							</div>
						</li>
						<li> 
							<div className="name">Ruby</div>
							<div className="progress p30">
								<div className="percentage" style={{ width: "30%" }}>
									<span className="percent">30%</span>
								</div>
								<span>30%</span>
							</div>
						</li>
						<li> 
							<div className="name">Bash</div>
							<div className="progress p80">
								<div className="percentage" style={{ width: "80%" }}>
									<span className="percent">80%</span>
								</div>
								<span>80%</span>
							</div>
						</li>
					</ul>
				</div>
			</div>
		</div>

		<div className="section skills">
			<div className="content">
				<div className="title">
					<div className="title_inner">Knowledge</div>
				</div>
				<div className="skills list">
					<ul>
						<li>
							<div className="name">git</div>
						</li>
						<li>
							<div className="name">Linux</div>
						</li>
						<li>
							<div className="name">Docker</div>
						</li>
						<li>
							<div className="name">Kubernetes</div>
						</li>
						<li>
							<div className="name">Cassandra</div>
						</li>
						<li>
							<div className="name">Datomic</div>
						</li>
						<li>
							<div className="name">PostgreSQL</div>
						</li>
						<li>
							<div className="name">mySQL</div>
						</li>
						<li>
							<div className="name">HBase</div>
						</li>
						<li>
							<div className="name">Oracle</div>
						</li>
						<li>
							<div className="name">Zookeeper</div>
						</li>
						<li>
							<div className="name">Kafka</div>
						</li>
						<li>
							<div className="name">RabbitMQ</div>
						</li>
						<li>
							<div className="name">om.next</div>
						</li>
						<li>
							<div className="name">React</div>
						</li>
						<li>
							<div className="name">Prometheus</div>
						</li>
						<li>
							<div className="name">SSH</div>
						</li>
						<li>
							<div className="name">NETCONF</div>
						</li>
						<li>
							<div className="name">gRPC</div>
						</li>
						<li>
							<div className="name">gNMI</div>
						</li>
						<li>
							<div className="name">YANG</div>
						</li>
						<li>
							<div className="name">OpenConfig</div>
						</li>
						<li>
							<div className="name">SNMP</div>
						</li>
						<li>
							<div className="name">TL1</div>
						</li>
						<li>
							<div className="name">eAPI</div>
						</li>
						<li>
							<div className="name">ICMP</div>
						</li>
						<li>
							<div className="name">DNS</div>
						</li>
						<li>
							<div className="name">HTTPS</div>
						</li>
						<li>
							<div className="name">HTTP2</div>
						</li>
						<li>
							<div className="name">REST APIs</div>
						</li>
						<li>
							<div className="name">GraphQL</div>
						</li>
					</ul>
				</div>
			</div>
		</div>
	</Fragment>
);

CV.propTypes = {
	/**
	 * _body: (test)(12)
	 */
	_body: PropTypes.node.isRequired,
};

CV.defaultProps = {};

export default CV;
