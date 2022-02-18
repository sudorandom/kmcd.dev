import PropTypes from 'prop-types';
import React, { Fragment } from 'react';


/**
 * The ResumeHeader component
 *
 * @disable-docs
 */
const ResumeHeader = ({ title, headerText }) => (
	<Fragment>
		<div className="section started">
			<div className="centrize full-width">
				<div className="vertical-center">
					<div className="started-content">
						<div className="h-title glitch-effect" data-text={ headerText }>{ headerText }</div>
						<div className="h-subtitle typing-subtitle">
							<p>Senior Software Engineer</p>
							<p>Based in Copenhagen</p>
							<p>This is my CV</p>
						</div>
						<span className="typed-subtitle"></span>
					</div>
				</div>
			</div>
			<a href="#" className="mouse_btn"><span className="ion ion-mouse"></span></a>
		</div>
	</Fragment>
);

ResumeHeader.defaultProps = {
	"headerText": "I am Kevin McDonald"
};

export default ResumeHeader;
